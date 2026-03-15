Unicode true

!define INFO_PROJECTNAME    "ts-escpos"
!define INFO_COMPANYNAME    "ts-escpos"
!define INFO_PRODUCTNAME    "ts-escpos"
!define PRODUCT_EXECUTABLE  "ts-escpos.exe"
!ifndef INFO_PRODUCTVERSION
  !define INFO_PRODUCTVERSION "1.0.0"
!endif

!define UNINST_KEY_NAME "${INFO_COMPANYNAME}${INFO_PRODUCTNAME}"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${UNINST_KEY_NAME}"

!define REQUEST_EXECUTION_LEVEL "admin"

RequestExecutionLevel "${REQUEST_EXECUTION_LEVEL}"

!include "MUI.nsh"
!include "WinVer.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

; Version Information
VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"
VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} (32-bit)"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"
VIAddVersionKey "LegalCopyright"  "Copyright 2024"

ManifestDPIAware true
ManifestSupportedOS all

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME} (32-bit)"
OutFile "..\..\bin\ts-escpos-386-installer.exe"
InstallDir "$PROGRAMFILES32\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"

ShowInstDetails show

Section
    SetShellVarContext all

    ; WebView2 Check
    ReadRegStr $0 HKLM "SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
    ${If} $0 == ""
        ; Check user level just in case
        ReadRegStr $0 HKCU "Software\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}" "pv"
        ${If} $0 == ""
            DetailPrint "Installing WebView2 Runtime..."
            InitPluginsDir
            CreateDirectory "$pluginsdir\webview2bootstrapper"
            SetOutPath "$pluginsdir\webview2bootstrapper"
            File "tmp\MicrosoftEdgeWebview2Setup.exe"
            ExecWait '"$pluginsdir\webview2bootstrapper\MicrosoftEdgeWebview2Setup.exe" /silent /install'
        ${EndIf}
    ${EndIf}

    SetOutPath $INSTDIR

    ; Install Binary
    ; We assume the binary is named ts-escpos-386.exe in the bin folder
    File "/oname=ts-escpos.exe" "..\..\bin\ts-escpos-386.exe"

    ; Shortcuts
    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$SMSTARTUP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    ; Uninstaller
    WriteUninstaller "$INSTDIR\uninstall.exe"

    WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${INFO_PRODUCTNAME} (32-bit)"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${INFO_PRODUCTVERSION}"
    WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
    WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${INFO_COMPANYNAME}"

    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD HKLM "${UNINST_KEY}" "EstimatedSize" "$0"
SectionEnd

Section "uninstall"
    SetShellVarContext all

    RMDir /r "$AppData\${INFO_PRODUCTNAME}" ; Remove data if desired, or keep it. Wails usually keeps it except for a specific subfolder.
    ; Wails default: RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"
    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"
    Delete "$SMSTARTUP\${INFO_PRODUCTNAME}.lnk"

    DeleteRegKey HKLM "${UNINST_KEY}"
SectionEnd

