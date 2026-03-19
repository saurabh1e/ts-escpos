Unicode true

# Define default constants if not provided via command line
!ifndef INFO_COMPANYNAME
  !define INFO_COMPANYNAME "saurabh1e"
!endif

!ifndef INFO_PRODUCTNAME
  !define INFO_PRODUCTNAME "ts-escpos"
!endif

!ifndef INFO_PRODUCTVERSION
  !define INFO_PRODUCTVERSION "0.0.0"
!endif

!ifndef INFO_COPYRIGHT
  !define INFO_COPYRIGHT "Copyright © 2026 saurabh1e"
!endif

!ifndef LITE_BINARY
  !error "LITE_BINARY must be defined"
!endif
!ifndef LITE_ARCH
  !error "LITE_ARCH must be defined"
!endif

!define APP_NAME "${INFO_PRODUCTNAME} Lite"
!define APP_EXE "ts-escpos-lite.exe"
!define UNINST_KEY "ts-escpos-lite"

# Set output file name based on LITE_ARCH
!if "${LITE_ARCH}" == "amd64"
  !define ARCH_SUFFIX "amd64"
!else
  !define ARCH_SUFFIX "386"
!endif

Name "${APP_NAME} (${LITE_ARCH})"
OutFile "..\..\bin\${INFO_PRODUCTNAME}-lite-${ARCH_SUFFIX}-setup.exe"
InstallDir "$LOCALAPPDATA\${INFO_COMPANYNAME}\${APP_NAME}"
RequestExecutionLevel user

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${APP_NAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${APP_NAME}"

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

  ; Protocol Handler
  DetailPrint "Registering ts-escpos URI Scheme..."
  DeleteRegKey HKCU "Software\Classes\ts-escpos"
  WriteRegStr HKCU "Software\Classes\ts-escpos" "" "URL:ts-escpos Protocol"
  WriteRegStr HKCU "Software\Classes\ts-escpos" "URL Protocol" ""
  WriteRegStr HKCU "Software\Classes\ts-escpos\DefaultIcon" "" "$INSTDIR\${APP_EXE},0"
  WriteRegStr HKCU "Software\Classes\ts-escpos\shell" "" ""
  WriteRegStr HKCU "Software\Classes\ts-escpos\shell\open" "" ""
  WriteRegStr HKCU "Software\Classes\ts-escpos\shell\open\command" "" '"$INSTDIR\${APP_EXE}" "%1"'
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

