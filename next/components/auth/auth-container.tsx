/** Shared centered shell for the authentication pages. */
export function AuthContainer({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="mx-auto w-full max-w-lg px-4 py-16">
      <header className="text-center">
        <h1 className="font-heading text-3xl font-semibold tracking-tight text-balance">
          {title}
        </h1>
        {description ? (
          <p className="mt-2 text-muted-foreground text-balance">
            {description}
          </p>
        ) : null}
      </header>
      {children}
    </div>
  );
}
