; ==============================================================================
; Nodal Windows Installer Script
; Publisher : Nodal Open Source Project
; Product   : Nodal - Native Windows 11 DNS Switcher (tray utility)
; License   : MIT (free of charge, no subscription, no license key)
; Requires  : Inno Setup 6.3 or newer  (https://jrsoftware.org/isdl.php)
;
; Compile:  iscc installer\setup.iss
;           iscc /DMyAppVersion=1.2.0 installer\setup.iss
; ==============================================================================

#define MyAppName "Nodal"
#ifndef MyAppVersion
  #define MyAppVersion "1.0.0"
#endif
#ifndef MyAppVersionFull
  #define MyAppVersionFull "1.0.0.0"
#endif
#define MyAppPublisher "Nodal Open Source Project"
#define MyAppURL "https://github.com/uswuth/Nodal"
#define MyAppExeName "nodal.exe"
#define MyAppDescription "Native Windows 11 DNS Switcher"

#if VER < EncodeVer(6,3,0)
  #error Nodal setup.iss requires Inno Setup 6.3 or newer (x64compatible architecture identifiers).
#endif

[Setup]
AppId={{D81E48A9-9F7C-4C67-B9F2-C5E3B21E9021}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppVerName={#MyAppName} {#MyAppVersion} - Free DNS Switcher
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}/issues
AppUpdatesURL={#MyAppURL}/releases
AppContact={#MyAppURL}/issues
AppComments=Free and open source. No telemetry, no data collection, no license key.
AppReadmeFile={app}\README.md
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
DisableProgramGroupPage=yes
DisableWelcomePage=no
AllowNoIcons=yes
LicenseFile=..\TERMS_AND_POLICY.md
InfoBeforeFile=..\PRIVACY_POLICY.md
OutputDir=..\dist
OutputBaseFilename=Nodal-Setup-{#MyAppVersion}
Compression=lzma2/ultra64
SolidCompression=yes
WizardStyle=modern
MinVersion=10.0
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
AppMutex=Local\Nodal_SingleInstance_Mutex
UninstallDisplayName={#MyAppName} - {#MyAppDescription}
UninstallDisplayIcon={app}\{#MyAppExeName}
VersionInfoVersion={#MyAppVersionFull}
VersionInfoCompany={#MyAppPublisher}
VersionInfoDescription={#MyAppName} Setup - {#MyAppDescription}
VersionInfoProductName={#MyAppName}
VersionInfoProductVersion={#MyAppVersion}
VersionInfoCopyright=Copyright (C) 2026 {#MyAppPublisher}. MIT License.

[Messages]
WelcomeLabel2=This wizard will install [name/ver] on your computer.%n%nPublisher: {#MyAppPublisher}%nDescription: {#MyAppDescription} (system tray utility)%nCost: Free of charge - no subscription, no license key, no account required%n%nNodal collects no personal data and sends no telemetry. It changes your network adapter's DNS resolver settings only when you pick a profile from the tray flyout.

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked
Name: "startupicon"; Description: "Start Nodal automatically when I sign in to Windows"; GroupDescription: "Startup:"

[Files]
; Tray application (no console window, no bundled third-party binaries)
Source: "..\bin\{#MyAppExeName}"; DestDir: "{app}"; Flags: ignoreversion
; Terms, privacy policy, license and usage documentation shipped with the app
Source: "..\TERMS_AND_POLICY.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\PRIVACY_POLICY.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\docs\installation.md"; DestDir: "{app}\docs"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"
Name: "{autoprograms}\{cm:UninstallProgram,{#MyAppName}}"; Filename: "{uninstallexe}"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon

[Run]
; Stage: privileged worker (scheduled task "Nodal" -> nodal.exe --worker).
; Skipped for per-user installs; Nodal then asks for elevation the first time you switch DNS.
Filename: "{app}\{#MyAppExeName}"; Parameters: "--register-task"; StatusMsg: "Registering the privileged DNS worker..."; Flags: runhidden ignoreerrors; Check: IsAdminInstallMode
; Stage: autostart preference, persisted to config.toml and the HKCU Run key.
Filename: "{app}\{#MyAppExeName}"; Parameters: "--enable-autostart"; StatusMsg: "Enabling launch on Windows sign-in..."; Flags: runhidden ignoreerrors; Tasks: startupicon
; Stage: first launch (tray icon appears next to the clock).
Filename: "{app}\{#MyAppExeName}"; Description: "{cm:LaunchProgram,{#StringChange(MyAppName, '&', '&&')}}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: filesandordirs; Name: "{localappdata}\Nodal"

[Code]
function InitializeUninstall(): Boolean;
var
  ResultCode: Integer;
  ConfigDir: String;
  TaskRegistered: Boolean;
begin
  Result := True;

  // Stop a running tray instance so no file stays locked during removal.
  Exec(ExpandConstant('{sys}\taskkill.exe'), '/f /im {#MyAppExeName}', '', SW_HIDE, ewWaitUntilTerminated, ResultCode);

  // Step 1: hand DNS back to Windows. The executable restores the automatic (DHCP-provided)
  // resolvers on every adapter and elevates itself when this uninstall is not already elevated.
  // It only exits once the work is done, so no override survives the removal of its files.
  if Exec(ExpandConstant('{app}\{#MyAppExeName}'), '--reset-dns', ExpandConstant('{app}'), SW_HIDE, ewWaitUntilTerminated, ResultCode) then
  begin
    if ResultCode <> 0 then
      SuppressibleMsgBox('The automatic (DHCP) DNS settings could not be restored on every network adapter.' + #13#10 + #13#10 +
        'Approve the administrator prompt and run this from an elevated Command Prompt:' + #13#10 +
        '"' + ExpandConstant('{app}\{#MyAppExeName}') + '" --reset-dns' + #13#10 + #13#10 +
        'Alternatively, select "Obtain DNS server address automatically" in the properties of the affected adapter.',
        mbInformation, MB_OK, IDOK);
  end;

  // Step 2: remove the privileged worker task. This needs elevation, so a per-user uninstall
  // explains the manual step - but only while the task is actually registered.
  TaskRegistered := False;
  if Exec(ExpandConstant('{sys}\schtasks.exe'), '/query /tn "{#MyAppName}"', '', SW_HIDE, ewWaitUntilTerminated, ResultCode) then
    TaskRegistered := (ResultCode = 0);

  if TaskRegistered then
  begin
    if (not Exec(ExpandConstant('{sys}\schtasks.exe'), '/delete /tn "{#MyAppName}" /f', '', SW_HIDE, ewWaitUntilTerminated, ResultCode)) or (ResultCode <> 0) then
      SuppressibleMsgBox('The privileged worker task "{#MyAppName}" could not be removed because this uninstall is not elevated.' + #13#10 + #13#10 +
        'To remove it manually, run the following in an elevated Command Prompt:' + #13#10 +
        'schtasks /delete /tn "{#MyAppName}" /f',
        mbInformation, MB_OK, IDOK);
  end;

  // Step 3: a startup entry pointing at an executable that is about to be deleted is always removed.
  if RegValueExists(HKEY_CURRENT_USER, 'Software\Microsoft\Windows\CurrentVersion\Run', '{#MyAppName}') then
    RegDeleteValue(HKEY_CURRENT_USER, 'Software\Microsoft\Windows\CurrentVersion\Run', '{#MyAppName}');

  // Step 4: the configuration and the DNS presets. An unattended uninstall answers this with the
  // default "Yes", which is what a clean removal expects; only the Nodal folders are deleted.
  ConfigDir := ExpandConstant('{userprofile}\.config\nodal');
  if DirExists(ConfigDir) then
  begin
    if SuppressibleMsgBox('Also delete your Nodal configuration and saved DNS presets?' + #13#10 + #13#10 + ConfigDir,
                          mbConfirmation, MB_YESNO, IDYES) = IDYES then
    begin
      DelTree(ConfigDir, True, True, True);
      ConfigDir := ExpandConstant('{userappdata}\nodal');
      if DirExists(ConfigDir) then
        DelTree(ConfigDir, True, True, True);
    end;
  end;
end;
