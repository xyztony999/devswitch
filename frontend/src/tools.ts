// 前端工具注册表：加新工具只需在此登记一条（图标另加 BrandLogo 分支）。
// 与 Go 侧 internal/models 的工具注册表概念对齐（D-01）。
import type { Tool } from './types'

export interface MirrorOption {
  label: string
  value: string
}

export interface ToolConfig {
  id: Tool
  label: string
  commands: string // hero 说明：该工具接管的命令
  blurb: string // hero 副标题
  mirrors: MirrorOption[]
  // 'major'：需要大版本号输入；'none'：latest 通道（下载条隐藏输入框）
  versionInput: 'major' | 'none'
  accent: string // 品牌色（hero 图标底染等）
}

const official: MirrorOption = { label: '官方源', value: 'official' }
const tuna: MirrorOption = { label: '清华源', value: 'tuna' }
const npmmirror: MirrorOption = { label: 'npmmirror', value: 'npmmirror' }
const tencent: MirrorOption = { label: '腾讯源', value: 'tencent' }
const officialCn: MirrorOption = { label: '官方源(国内)', value: 'official' }

export const TOOL_CONFIGS: ToolConfig[] = [
  {
    id: 'node',
    label: 'Node.js',
    commands: 'node / npm',
    blurb: 'JavaScript 运行时 · 版本切换对已打开的终端立即生效',
    mirrors: [official, npmmirror],
    versionInput: 'major',
    accent: '#8CC84B'
  },
  {
    id: 'java',
    label: 'Java',
    commands: 'java / javac',
    blurb: 'JDK · 同步维护 JAVA_HOME（终端钩子 + 用户环境变量）',
    mirrors: [official, tuna],
    versionInput: 'major',
    accent: '#E76F00'
  },
  {
    id: 'maven',
    label: 'Maven',
    commands: 'mvn',
    blurb: '构建工具 · 同步维护 MAVEN_HOME',
    mirrors: [official, tuna],
    versionInput: 'major',
    accent: '#C71A36'
  },
  {
    id: 'gradle',
    label: 'Gradle',
    commands: 'gradle',
    blurb: '构建工具 · 同步维护 GRADLE_HOME',
    mirrors: [official, tuna],
    versionInput: 'major',
    accent: '#1BA6C4'
  },
  {
    id: 'androidsdk',
    label: 'Android SDK',
    commands: 'adb / fastboot',
    blurb: 'Android 开发套件 · adb / fastboot 即时生效，同步维护 ANDROID_HOME',
    mirrors: [official, tencent],
    versionInput: 'none',
    accent: '#3DDC84'
  },
  {
    id: 'golang',
    label: 'Go',
    commands: 'go / gofmt',
    blurb: 'Go 工具链 · 同步维护 GOROOT（GOPATH/GOPROXY 不动）',
    mirrors: [officialCn],
    versionInput: 'none',
    accent: '#00ADD8'
  },
  {
    id: 'python',
    label: 'Python',
    commands: 'python / pip',
    blurb: 'Python · 完整 CPython 含 pip（python-build-standalone），不设 PYTHONHOME',
    mirrors: [npmmirror],
    versionInput: 'none',
    accent: '#3776AB'
  }
]

export const toolConfig = (id: Tool): ToolConfig =>
  TOOL_CONFIGS.find(t => t.id === id) || TOOL_CONFIGS[0]
