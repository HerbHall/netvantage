; SubNetree Scout Agent Installer
; Inno Setup 6+ script
;
; Build: iscc installer\inno\subnetree-scout.iss
; Requires SCOUT_VERSION env var (defaults to 0.0.0-dev)

#define MyAppName "SubNetree Scout Agent"
#define MyAppVersion GetEnv('SCOUT_VERSION')
#if MyAppVersion == ""
  #define MyAppVersion "0.0.0-dev"
#endif
#define MyAppPublisher "SubNetree"
#define MyAppURL "https://github.com/HerbHall/subnetree"
#define MyAppExeName "scout.exe"

[Setup]
AppId={{5E8F9C2A-7B3D-4A1E-B6F0-8D2C3E4A5B6C}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
DefaultDirName={autopf}\SubNetree\Scout
DefaultGroupName=SubNetree
LicenseFile=..\..\LICENSE
OutputDir=output
OutputBaseFilename=SubNetreeScout-{#MyAppVersion}-setup
Compression=lzma
SolidCompression=yes
WizardStyle=modern
MinVersion=10.0
PrivilegesRequired=admin
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\{#MyAppExeName}

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "addtopath"; Description: "Add Scout to system PATH"; GroupDescription: "Additional options:"

[Files]
Source: "bin\scout.exe"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\SubNetree Scout"; Filename: "{app}\{#MyAppExeName}"
Name: "{group}\Uninstall SubNetree Scout"; Filename: "{uninstallexe}"

[Registry]
Root: HKLM; Subkey: "SYSTEM\CurrentControlSet\Control\Session Manager\Environment"; \
    ValueType: expandsz; ValueName: "Path"; ValueData: "{olddata};{app}"; \
    Tasks: addtopath; Check: NeedsAddPath(ExpandConstant('{app}'))

[UninstallDelete]
Type: dirifempty; Name: "{app}"
Type: dirifempty; Name: "{autopf}\SubNetree"

[Code]
function NeedsAddPath(Param: string): Boolean;
var
  OrigPath: string;
begin
  if not RegQueryStringValue(HKLM,
    'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
    'Path', OrigPath)
  then begin
    Result := True;
    exit;
  end;
  { Check if the path is already present (case-insensitive) }
  Result := Pos(';' + Uppercase(Param) + ';', ';' + Uppercase(OrigPath) + ';') = 0;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  OrigPath: string;
  AppDir: string;
  P: Integer;
begin
  if CurUninstallStep = usPostUninstall then
  begin
    AppDir := ExpandConstant('{app}');
    if RegQueryStringValue(HKLM,
      'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
      'Path', OrigPath)
    then begin
      { Remove ;{app} from PATH }
      P := Pos(';' + Uppercase(AppDir), Uppercase(OrigPath));
      if P <> 0 then
      begin
        Delete(OrigPath, P, Length(';' + AppDir));
        RegWriteStringValue(HKLM,
          'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
          'Path', OrigPath);
      end;
      { Also check if it starts with {app}; }
      if Pos(Uppercase(AppDir) + ';', Uppercase(OrigPath)) = 1 then
      begin
        Delete(OrigPath, 1, Length(AppDir) + 1);
        RegWriteStringValue(HKLM,
          'SYSTEM\CurrentControlSet\Control\Session Manager\Environment',
          'Path', OrigPath);
      end;
    end;
  end;
end;
