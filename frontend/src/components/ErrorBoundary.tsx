import { Component, type ErrorInfo, type ReactNode } from 'react'
import Button from '@/components/ui/Button'

type ErrorBoundaryProps = {
  children: ReactNode
}

type ErrorBoundaryState = {
  error: Error | null
}

export default class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  state: ErrorBoundaryState = { error: null }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('App render error', error, info)
  }

  render() {
    if (!this.state.error) return this.props.children

    return (
      <div className="min-h-screen bg-navy-900 p-6 text-white">
        <section className="mx-auto mt-16 max-w-xl rounded-lg border border-danger/30 bg-danger/10 p-6">
          <h1 className="text-lg font-semibold">La pantalla falló al renderizar</h1>
          <p className="mt-2 text-sm text-slate-300">
            Recarga la página. Si vuelve a pasar, copia este error para revisarlo.
          </p>
          <pre className="mt-4 max-h-56 overflow-auto rounded-lg bg-navy-950 p-3 text-xs text-red-200">
            {this.state.error.message}
          </pre>
          <Button className="mt-4" onClick={() => window.location.reload()}>
            Recargar
          </Button>
        </section>
      </div>
    )
  }
}
