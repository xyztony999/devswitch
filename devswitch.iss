; DevSwitch Windows 安装程序脚本（Inno Setup 6）—— 2.0 起打包 Go 单文件二进制
;
; 编译：  ISCC devswitch.iss
; 版本：  CI 用 /DAppVersion=x.y.z 覆盖；本地默认取下行
; 架构：  /DArchMode=arm64 打包 arm64 二进制（默认 x64）；
;         二进制来源用 /DBinSource=dist\bin\devswitch-arm64.exe 覆盖
; 命名：  产物统一小写 devswitch 前缀（与 deb/rpm 一致）；
;         向导与控制面板显示名仍为 DevSwitch（AppName）
; 语言：  installer\ChineseSimplified.isl 存在时提供简中向导（CI 会自动下载），
;         缺失时使用英文向导
#ifndef AppVersion
#define AppVersion "2.0.0"
#endif
#ifndef ArchMode
#define ArchMode "x64compatible"
#endif
#ifndef BinSource
#define BinSource "dist\bin\devswitch-x64.exe"
#endif
#if ArchMode == "arm64"
#define ArchSuffix "-arm64"
#else
#define ArchSuffix "-x64"
#endif

[Setup]
AppId={{7C4F1B8E-2D63-4A95-9E17-6F0B3D8A2C44}
AppName=DevSwitch
AppVersion={#AppVersion}
AppPublisher=xyztony999
AppPublisherURL=https://github.com/xyztony999/devswitch
AppUpdatesURL=https://github.com/xyztony999/devswitch/releases
; 纯用户级安装，全程无需管理员
PrivilegesRequired=lowest
DefaultDirName={localappdata}\devswitch\app
; 总是使用上面的默认目录，不受任何历史安装位置影响
UsePreviousAppDir=no
DirExistsWarning=no
DisableProgramGroupPage=yes
DefaultGroupName=DevSwitch
UninstallDisplayName=DevSwitch
UninstallDisplayIcon={app}\devswitch.ico
SetupIconFile=share\icons\devswitch.ico
OutputDir=dist
OutputBaseFilename=devswitch-setup-{#AppVersion}{#ArchSuffix}
; arm64：生成原生 ARM64 安装器；x64compatible：x64 安装器（ARM64 设备可模拟运行）
ArchitecturesInstallIn64BitMode={#ArchMode}
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
MinVersion=10.0

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"
#if FileExists(SourcePath + "\installer\ChineseSimplified.isl")
Name: "chinesesimplified"; MessagesFile: "installer\ChineseSimplified.isl"
#endif

[Tasks]
Name: "desktopicon"; Description: "创建桌面快捷方式 / Create desktop shortcut"; GroupDescription: "附加选项 / Additional options:"; Flags: unchecked

[Files]
; Go 单文件二进制（GUI 与 CLI 同体；首次运行自动完成用户级配置）
Source: "{#BinSource}"; DestDir: "{app}"; DestName: "devswitch.exe"; Flags: ignoreversion
Source: "share\icons\devswitch.ico"; DestDir: "{app}"; Flags: ignoreversion

[InstallDelete]
; 1.x（Python 版）装到 {app}\devswitch\，升级时清掉旧布局
Type: filesandordirs; Name: "{app}\devswitch"

[Icons]
; 无参数启动即进入图形界面（见 cmd/devswitch 默认行为）
Name: "{autoprograms}\DevSwitch"; Filename: "{app}\devswitch.exe"; Comment: "Node / Java / Maven / Gradle 版本切换器"; IconFilename: "{app}\devswitch.ico"
Name: "{autodesktop}\DevSwitch"; Filename: "{app}\devswitch.exe"; Comment: "Node / Java / Maven / Gradle 版本切换器"; IconFilename: "{app}\devswitch.ico"; Tasks: desktopicon

[Run]
; 首次配置（shim / 用户 PATH / shell 钩子 / 扫描本机版本），静默执行
Filename: "{app}\devswitch.exe"; Parameters: "scan"; WorkingDir: "{app}"; StatusMsg: "正在配置 DevSwitch（shim / PATH / 钩子）……"; Flags: runhidden waituntilterminated
Filename: "{app}\devswitch.exe"; Parameters: "gui"; Description: "运行 DevSwitch / Launch DevSwitch"; WorkingDir: "{app}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
; 清理 shim / 用户 PATH / JAVA_HOME 等用户变量 / shell 钩子（程序文件由向导删除）
Filename: "{app}\devswitch.exe"; Parameters: "cleanup"; WorkingDir: "{app}"; RunOnceId: "DevSwitchCleanup"; Flags: runhidden waituntilterminated

[UninstallDelete]
Type: filesandordirs; Name: "{app}"
