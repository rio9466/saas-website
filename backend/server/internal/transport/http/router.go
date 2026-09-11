package http

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
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

	if deps.AdminAuth == nil && deps.UserClient == nil && deps.UserAdmin == nil {
		return r
	}

	authenticate := middleware.Authenticate(deps.Tokens, deps.Sessions, deps.AdminAuth)

	// --- business-user public/auth/me surface ------------------------------
	if deps.UserClient != nil {
		userAuthHandlers := handler.NewUserAuthHandlers(deps.UserClient, deps.UserCookie, deps.TrustedOrigins)
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

	return r
}
