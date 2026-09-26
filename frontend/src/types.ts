export type Tool = 'node' | 'java' | 'maven' | 'gradle'
export type ToolPage = Tool | 'doctor'

export interface Runtime {
  tool: Tool
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
  installing: boolean
  current: Partial<Record<Tool, Runtime | null>>
  runtimes: Runtime[]
  issues: Issue[]
  selected: string
  flash?: { text: string; kind?: string } | null
  paths: { localBin?: string; config?: string }
}
