Unicode true

!include "wails_tools.nsh"

!ifndef LITE_BINARY
  !error "LITE_BINARY must be defined"
!endif
!ifndef LITE_ARCH
  !error "LITE_ARCH must be defined"
!endif

!define APP_NAME "${INFO_PRODUCTNAME} Lite"
!define APP_EXE "ts-escpos-lite.exe"
!define UNINST_KEY "ts-escpos-lite"

Name "${APP_NAME} (${LITE_ARCH})"
OutFile "..\..\bin\ts-escpos-lite-${LITE_ARCH}-installer.exe"
InstallDir "$LOCALAPPDATA\${INFO_COMPANYNAME}\${APP_NAME}"
RequestExecutionLevel user

!include "MUI2.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"

  ; Install the executable (rename to standard name)
  File "/oname=${APP_EXE}" "${LITE_BINARY}"

  ; Create Uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Desktop Shortcut (Optional - omitted for headless/lite)
  ; CreateShortCut "$DESKTOP\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}"

  ; Start Menu Shortcut
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\${APP_NAME}.lnk" "$INSTDIR\${APP_EXE}"
  CreateShortCut "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"

  ; Registry - Auto Start
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "${UNINST_KEY}" "$INSTDIR\${APP_EXE}"

  ; Registry - Add/Remove Programs
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY}" "DisplayName" "${APP_NAME}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${APP_EXE}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
SectionEnd

Section "Uninstall"
  ; Remove Files
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\uninstall.exe"

  ; Remove Shortcuts
  ; Delete "$DESKTOP\${APP_NAME}.lnk"
  RMDir /r "$SMPROGRAMS\${APP_NAME}"

  ; Remove Directories
  RMDir "$INSTDIR"

  ; Remove Registry Keys
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "${UNINST_KEY}"
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY}"
SectionEnd

