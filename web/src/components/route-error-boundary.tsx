import { useRouteError, isRouteErrorResponse } from 'react-router-dom'
import { RefreshCw, AlertTriangle } from 'lucide-react'

export function RouteErrorBoundary() {
  const error = useRouteError()

  const isChunkError =
    error instanceof TypeError &&
    error.message.includes('dynamically imported module')

  function handleRefresh() {
    window.location.reload()
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-8">
      <div className="max-w-md text-center space-y-4">
        <AlertTriangle className="mx-auto h-12 w-12 text-yellow-500" />
        {isChunkError ? (
          <>
            <h1 className="text-xl font-semibold">App Updated</h1>
            <p className="text-muted-foreground">
              A new version was deployed while you had the page open.
              Refresh to load the latest version.
            </p>
            <button
              onClick={handleRefresh}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh Now
            </button>
          </>
        ) : (
          <>
            <h1 className="text-xl font-semibold">Something Went Wrong</h1>
            <p className="text-muted-foreground">
              {isRouteErrorResponse(error)
                ? `${error.status}: ${error.statusText}`
                : error instanceof Error
                  ? error.message
                  : 'An unexpected error occurred.'}
            </p>
            <button
              onClick={handleRefresh}
              className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <RefreshCw className="h-4 w-4" />
              Try Again
            </button>
          </>
        )}
      </div>
    </div>
  )
}
