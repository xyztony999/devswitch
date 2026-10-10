declare global {
  interface Window {
    webkit?: {
      messageHandlers?: {
        devswitch?: { postMessage: (msg: string) => void }
      }
    }
    pywebview?: { api?: { send?: (msg: string) => Promise<unknown> | void } }
  }
}

export function send(msg: Record<string, unknown>): void {
  const payload = JSON.stringify(msg)
  const handler = window.webkit?.messageHandlers?.devswitch
  if (handler) {
    handler.postMessage(payload)
    return
  }
  const api = window.pywebview?.api
  if (api?.send) {
    void api.send(payload)
    return
  }
  // pywebview injects its bridge shortly after load; retry once it is ready.
  window.addEventListener(
    'pywebviewready',
    () => {
      void window.pywebview?.api?.send?.(payload)
    },
    { once: true }
  )
}

export function sourceText(source: string): string {
  if (source === 'system') return '系统安装'
  if (source === 'imported') return '手动导入'
  if (source === 'env') return 'ANDROID_HOME 指向'
  if (source === 'path') return 'PATH 发现'
  if (source === 'downloaded') return '本工具下载'
  return '本机目录'
}

export function issueLabel(level: string): string {
  if (level === 'error') return '错误'
  if (level === 'info') return '提示'
  return '注意'
}
