; DevSwitch Windows 安装程序脚本（Inno Setup 6）
;
; 编译：  ISCC devswitch.iss
; 版本：  CI 可用 /DAppVersion=x.y.z 覆盖；本地默认取下行（与 __init__.py 同步维护）
; 架构：  /DArchMode=arm64 产出原生 ARM64 安装器（默认 x64compatible）
; 语言：  installer\ChineseSimplified.isl 存在时提供简中向导（CI 会自动下载），
;         缺失时使用英文向导
#ifndef AppVersion
#define AppVersion "1.1.0"
#endif
#ifndef ArchMode
#define ArchMode "x64compatible"
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
OutputBaseFilename=DevSwitch-Setup-{#AppVersion}{#ArchSuffix}
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
; 运行时包：{app}\devswitch（python 包，运行 devswitch install --in-place 就地配置）
Source: "devswitch\*"; DestDir: "{app}\devswitch"; Excludes: "__pycache__,*.pyc"; Flags: recursesubdirs ignoreversion
Source: "share\icons\devswitch.ico"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
; 启动器在 [Run] 阶段由 devswitch install 生成于 {localappdata}\devswitch\bin
Name: "{autoprograms}\DevSwitch"; Filename: "{app}\..\bin\devswitch-gui.cmd"; WorkingDir: "{app}"; Comment: "Node / Java 版本切换器"; IconFilename: "{app}\devswitch.ico"
Name: "{autodesktop}\DevSwitch"; Filename: "{app}\..\bin\devswitch-gui.cmd"; WorkingDir: "{app}"; Comment: "Node / Java 版本切换器"; IconFilename: "{app}\devswitch.ico"; Tasks: desktopicon

[Run]
; 核心配置（shim、启动器、用户 PATH、shell 钩子、GUI 依赖）复用包内安装逻辑
Filename: "{cmd}"; Parameters: "/C py -3 -m devswitch install --in-place"; WorkingDir: "{app}"; StatusMsg: "正在配置 DevSwitch（shim / PATH / 钩子）……"; Check: PyLauncherExists
Filename: "{cmd}"; Parameters: "/C python -m devswitch install --in-place"; WorkingDir: "{app}"; StatusMsg: "正在配置 DevSwitch（shim / PATH / 钩子）……"; Check: NotPyLauncherExists
Filename: "{app}\..\bin\devswitch-gui.cmd"; Description: "运行 DevSwitch / Launch DevSwitch"; WorkingDir: "{app}"; Flags: nowait postinstall skipifsilent

[UninstallRun]
; 清理 shim / 启动器 / 用户 PATH / JAVA_HOME / 钩子（程序文件由向导自己删除）
Filename: "{cmd}"; Parameters: "/C py -3 -m devswitch uninstall --keep-app-files"; WorkingDir: "{app}"; RunOnceId: "DevSwitchCleanup"; Check: PyLauncherExists
Filename: "{cmd}"; Parameters: "/C python -m devswitch uninstall --keep-app-files"; WorkingDir: "{app}"; RunOnceId: "DevSwitchCleanupPy"; Check: NotPyLauncherExists

[UninstallDelete]
; 连同运行期产生的 __pycache__ 等一并清掉
Type: filesandordirs; Name: "{app}"

[Code]
function PyLauncherExists(): Boolean;
begin
  Result := FileExists(GetEnv('SystemRoot') + '\py.exe');
end;

function NotPyLauncherExists(): Boolean;
begin
  Result := not PyLauncherExists();
end;

function PythonOk(): Boolean;
var
  rc: Integer;
begin
  Result := False;
  if Exec(GetEnv('SystemRoot') + '\System32\cmd.exe',
          '/C py -3 -c "import sys; sys.exit(0 if sys.version_info >= (3, 8) else 1)"',
          '', SW_HIDE, ewWaitUntilTerminated, rc) and (rc = 0) then
  begin
    Result := True;
    Exit;
  end;
  if Exec(GetEnv('SystemRoot') + '\System32\cmd.exe',
          '/C python -c "import sys; sys.exit(0 if sys.version_info >= (3, 8) else 1)"',
          '', SW_HIDE, ewWaitUntilTerminated, rc) and (rc = 0) then
  begin
    Result := True;
    Exit;
  end;
end;

function InitializeSetup(): Boolean;
var
  err: Integer;
begin
  Result := PythonOk();
  if not Result then
  begin
    if MsgBox(
      'DevSwitch 需要 Python 3.8+，当前没有检测到。' + #13#10 + #13#10 +
      '是否现在打开下载页面？安装 Python 后请重新运行本安装程序。',
      mbConfirmation, MB_YESNO) = IDYES then
      ShellExec('open', 'https://www.python.org/downloads/', '', '', SW_SHOWNORMAL, ewNoWait, err);
  end;
end;
