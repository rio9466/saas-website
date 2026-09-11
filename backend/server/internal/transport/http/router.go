package http

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
	"github.com/rio9466/easy-admin/server/internal/transport/http/handler"
	"github.com/rio9466/easy-admin/server/internal/transport/http/middleware"
)

// Dependencies are the runtime collaborators required by the HTTP surface.
type Dependencies struct {
	Logger         *slog.Logger
	ReadyChecker   handler.ReadyChecker
	AdminAuth      handler.AdminAuthService
	UserClient     handler.UserClientService
	UserAdmin      handler.UserAdminService
	Content        handler.ContentService
	Tokens         middleware.TokenParser
	Sessions       middleware.SessionGetter
	UserTokens     middleware.TokenParser
	UserSessions   middleware.SessionGetter
	Cookie         handler.AuthCookieSettings
	UserCookie     handler.UserCookieSettings
	TrustedOrigins []string
}

// NewRouter builds the Gin engine with middleware, health, admin, and
// business-user APIs.
func NewRouter(deps Dependencies) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(logger))

	r.GET("/healthz", handler.Healthz)
	r.GET("/readyz", handler.Readyz(deps.ReadyChecker))

	// --- public content surface (no authentication) ------------------------
	if deps.Content != nil {
		contentPublic := handler.NewContentPublicHandlers(deps.Content)
		r.GET("/api/v1/public/navigation", contentPublic.Navigation)
		r.GET("/api/v1/public/home", contentPublic.Home)
		r.GET("/api/v1/public/features", contentPublic.Features)
		r.GET("/api/v1/public/pricing", contentPublic.Pricing)
		r.GET("/api/v1/public/pages", contentPublic.Pages)
		r.GET("/api/v1/public/pages/:slug", contentPublic.Page)
		r.GET("/api/v1/public/docs", contentPublic.Docs)
		r.GET("/api/v1/public/docs/:slug", contentPublic.Doc)
	}

	if deps.AdminAuth == nil && deps.UserClient == nil && deps.UserAdmin == nil {
		return r
	}

	authenticate := middleware.Authenticate(deps.Tokens, deps.Sessions, deps.AdminAuth)

	// --- business-user public/auth/me surface ------------------------------
	if deps.UserClient != nil {
		userAuthHandlers := handler.NewUserAuthHandlers(deps.UserClient, deps.Content, deps.UserCookie, deps.TrustedOrigins)
		userAuthenticate := middleware.AuthenticateUser(deps.UserTokens, deps.UserSessions, deps.UserClient)

		r.GET("/api/v1/public/settings", userAuthHandlers.PublicSettings)

		userAuth := r.Group("/api/v1/auth")
		{
			userAuth.POST("/register", userAuthHandlers.Register)
			userAuth.POST("/verify-email", userAuthHandlers.VerifyEmail)
			userAuth.POST("/resend-verification", userAuthHandlers.ResendVerification)
			userAuth.POST("/login", userAuthHandlers.Login)
			userAuth.POST("/refresh", userAuthHandlers.Refresh)
			userAuth.POST("/logout", userAuthenticate, userAuthHandlers.Logout)
		}

		me := r.Group("/api/v1/me")
		me.Use(userAuthenticate)
		{
			me.GET("", userAuthHandlers.Me)
		}
	}

	// --- administrator surface --------------------------------------------
	if deps.AdminAuth == nil {
		return r
	}

	authHandlers := handler.NewAuthHandlers(deps.AdminAuth, deps.Cookie, deps.TrustedOrigins)
	adminHandlers := handler.NewAdministratorHandlers(deps.AdminAuth)
	roleHandlers := handler.NewRoleHandlers(deps.AdminAuth)
	auditHandlers := handler.NewAuditHandlers(deps.AdminAuth)

	v1 := r.Group("/api/v1/admin")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandlers.Login)
			auth.POST("/refresh", authHandlers.Refresh)
			auth.POST("/logout", authenticate, authHandlers.Logout)
		}

		me := v1.Group("")
		me.Use(authenticate)
		{
			me.GET("/me", authHandlers.Me)
			me.PATCH("/me", authHandlers.UpdateMe)
			me.POST("/me/password", authHandlers.ChangePassword)
		}

		admins := v1.Group("/administrators")
		admins.Use(authenticate)
		{
			admins.GET("", middleware.RequirePermission(adminauth.PermAdminUserRead), adminHandlers.List)
			admins.POST("", middleware.RequirePermission(adminauth.PermAdminUserCreate), adminHandlers.Create)
			admins.GET("/:id", middleware.RequirePermission(adminauth.PermAdminUserRead), adminHandlers.Get)
			admins.PATCH("/:id", middleware.RequirePermission(adminauth.PermAdminUserUpdate), adminHandlers.Update)
			admins.POST("/:id/enable", middleware.RequirePermission(adminauth.PermAdminUserDisable), adminHandlers.Enable)
			admins.POST("/:id/disable", middleware.RequirePermission(adminauth.PermAdminUserDisable), adminHandlers.Disable)
			admins.POST("/:id/reset-password", middleware.RequirePermission(adminauth.PermAdminUserResetPwd), adminHandlers.ResetPassword)
			admins.PUT("/:id/roles", middleware.RequirePermission(adminauth.PermAdminUserAssignRole), adminHandlers.AssignRoles)
		}

		roles := v1.Group("/roles")
		roles.Use(authenticate)
		{
			roles.GET("", middleware.RequirePermission(adminauth.PermAdminRoleRead), roleHandlers.ListRoles)
			roles.POST("", middleware.RequirePermission(adminauth.PermAdminRoleManage), roleHandlers.CreateRole)
			roles.GET("/:id", middleware.RequirePermission(adminauth.PermAdminRoleRead), roleHandlers.GetRole)
			roles.PATCH("/:id", middleware.RequirePermission(adminauth.PermAdminRoleManage), roleHandlers.UpdateRole)
			roles.PUT("/:id/permissions", middleware.RequirePermission(adminauth.PermAdminRoleManage), roleHandlers.ReplacePermissions)
		}

		v1.GET("/permissions", authenticate, middleware.RequirePermission(adminauth.PermAdminRoleRead), roleHandlers.ListPermissions)

		audit := v1.Group("/audit-events")
		audit.Use(authenticate)
		{
			audit.GET("", middleware.RequirePermission(adminauth.PermAuditLogRead), auditHandlers.List)
			audit.GET("/:id", middleware.RequirePermission(adminauth.PermAuditLogRead), auditHandlers.Get)
		}

		// --- business-user management (administrator surface) --------------
		if deps.UserAdmin != nil {
			userAdminHandlers := handler.NewUserAdminHandlers(deps.UserAdmin)
			userLevelHandlers := handler.NewUserLevelHandlers(deps.UserAdmin)
			systemSettingsHandlers := handler.NewSystemSettingsHandlers(deps.UserAdmin)

			users := v1.Group("/users")
			users.Use(authenticate)
			{
				users.GET("", middleware.RequirePermission(adminauth.PermCustomerRead), userAdminHandlers.List)
				users.POST("", middleware.RequirePermission(adminauth.PermCustomerCreate), userAdminHandlers.Create)
				users.GET("/:id", middleware.RequirePermission(adminauth.PermCustomerRead), userAdminHandlers.Get)
				users.PATCH("/:id", middleware.RequirePermission(adminauth.PermCustomerUpdate), userAdminHandlers.Update)
				users.POST("/:id/enable", middleware.RequirePermission(adminauth.PermCustomerDisable), userAdminHandlers.Enable)
				users.POST("/:id/disable", middleware.RequirePermission(adminauth.PermCustomerDisable), userAdminHandlers.Disable)
				users.POST("/:id/reset-password", middleware.RequirePermission(adminauth.PermCustomerResetPwd), userAdminHandlers.ResetPassword)
				users.POST("/:id/points-adjust", middleware.RequirePermission(adminauth.PermCustomerPoints), userAdminHandlers.AdjustPoints)
				users.GET("/:id/point-transactions", middleware.RequirePermission(adminauth.PermCustomerPoints), userAdminHandlers.ListPointTransactions)
				users.POST("/:id/level", middleware.RequirePermission(adminauth.PermCustomerLevelAssign), userAdminHandlers.AssignLevel)
			}

			levels := v1.Group("/user-levels")
			levels.Use(authenticate)
			{
				levels.GET("", middleware.RequirePermission(adminauth.PermUserLevelRead), userLevelHandlers.List)
				levels.POST("", middleware.RequirePermission(adminauth.PermUserLevelManage), userLevelHandlers.Create)
				levels.GET("/:id", middleware.RequirePermission(adminauth.PermUserLevelRead), userLevelHandlers.Get)
				levels.PATCH("/:id", middleware.RequirePermission(adminauth.PermUserLevelManage), userLevelHandlers.Update)
			}

			settings := v1.Group("/system-settings")
			settings.Use(authenticate)
			{
				settings.GET("", middleware.RequirePermission(adminauth.PermSystemSettingsRead), systemSettingsHandlers.Get)
				settings.PUT("", middleware.RequirePermission(adminauth.PermSystemSettingsManage), systemSettingsHandlers.Update)
			}
		}
	}

	// --- administrator content management ---------------------------------
	if deps.Content != nil {
		contentHandlers := handler.NewContentAdminHandlers(deps.Content)

		siteContent := v1.Group("/site-settings")
		siteContent.Use(authenticate)
		{
			siteContent.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.GetSiteSettings)
			siteContent.PUT("", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdateSiteSettings)
		}

		navigation := v1.Group("/navigation-items")
		navigation.Use(authenticate)
		{
			navigation.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.ListNavigationItems)
			navigation.POST("", middleware.RequirePermission(content.PermissionManage), contentHandlers.CreateNavigationItem)
			navigation.PATCH("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdateNavigationItem)
			navigation.DELETE("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.DeleteNavigationItem)
		}

		homeSections := v1.Group("/home-sections")
		homeSections.Use(authenticate)
		{
			homeSections.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.ListHomeSections)
			homeSections.POST("", middleware.RequirePermission(content.PermissionManage), contentHandlers.CreateHomeSection)
			homeSections.PATCH("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdateHomeSection)
			homeSections.DELETE("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.DeleteHomeSection)
		}

		features := v1.Group("/features")
		features.Use(authenticate)
		{
			features.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.ListFeatures)
			features.POST("", middleware.RequirePermission(content.PermissionManage), contentHandlers.CreateFeature)
			features.PATCH("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdateFeature)
			features.DELETE("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.DeleteFeature)
		}

		pricingPlans := v1.Group("/pricing-plans")
		pricingPlans.Use(authenticate)
		{
			pricingPlans.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.ListPricingPlans)
			pricingPlans.POST("", middleware.RequirePermission(content.PermissionManage), contentHandlers.CreatePricingPlan)
			pricingPlans.PATCH("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdatePricingPlan)
			pricingPlans.DELETE("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.DeletePricingPlan)
		}

		pages := v1.Group("/pages")
		pages.Use(authenticate)
		{
			pages.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.ListPages)
			pages.POST("", middleware.RequirePermission(content.PermissionManage), contentHandlers.CreatePage)
			pages.GET("/:id", middleware.RequirePermission(content.PermissionRead), contentHandlers.GetPage)
			pages.PATCH("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdatePage)
			pages.DELETE("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.DeletePage)
		}

		docCategories := v1.Group("/doc-categories")
		docCategories.Use(authenticate)
		{
			docCategories.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.ListDocCategories)
			docCategories.POST("", middleware.RequirePermission(content.PermissionManage), contentHandlers.CreateDocCategory)
			docCategories.PATCH("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdateDocCategory)
			docCategories.DELETE("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.DeleteDocCategory)
		}

		docArticles := v1.Group("/doc-articles")
		docArticles.Use(authenticate)
		{
			docArticles.GET("", middleware.RequirePermission(content.PermissionRead), contentHandlers.ListDocArticles)
			docArticles.POST("", middleware.RequirePermission(content.PermissionManage), contentHandlers.CreateDocArticle)
			docArticles.GET("/:id", middleware.RequirePermission(content.PermissionRead), contentHandlers.GetDocArticle)
			docArticles.PATCH("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.UpdateDocArticle)
			docArticles.DELETE("/:id", middleware.RequirePermission(content.PermissionManage), contentHandlers.DeleteDocArticle)
		}
	}

	return r
}
