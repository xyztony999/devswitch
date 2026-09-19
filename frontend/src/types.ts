export type ToolPage = 'node' | 'java' | 'doctor'

export interface Runtime {
  tool: 'node' | 'java'
  version: string
  major: string
  home: string
  binary: string
  source: string
  label: string
  vendor: string
}

export interface Issue {
  level: 'error' | 'warn' | 'info' | string
  code: string
  message: string
}

export interface AppState {
  page: ToolPage
  version: string
  current: { node?: Runtime | null; java?: Runtime | null }
  runtimes: Runtime[]
  issues: Issue[]
  selected: string
  flash?: { text: string; kind?: string } | null
  paths: { localBin?: string; config?: string }
}
