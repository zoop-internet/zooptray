!include "MUI2.nsh"
!include "FileFunc.nsh"

!ifndef VERSION
  !define VERSION "1.0.0"
!endif

Name "Zoop One"
OutFile "Zoop-One-Setup-${VERSION}.exe"
InstallDir "$PROGRAMFILES64\Zoop One"
RequestExecutionLevel admin

!define PRODUCT_NAME "Zoop One"
!define PRODUCT_PUBLISHER "Zoop Network"
!define PRODUCT_URL "https://zoopnetwork.com"
!define UNINSTALL_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "EULA.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_LANGUAGE "English"

Section "-Install"
  SetOutPath "$INSTDIR"
  File "staging\zoop-one.exe"
  File "icon.ico"

  SetOutPath "$APPDATA\Zoop"
  File "staging\config.json"

  WriteUninstaller "$INSTDIR\uninstall.exe"

  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayName" "${PRODUCT_NAME}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "UninstallString" "$INSTDIR\uninstall.exe"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayIcon" "$INSTDIR\zoop-one.exe"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegStr HKLM "${UNINSTALL_KEY}" "URLInfoAbout" "${PRODUCT_URL}"
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "NoRepair" 1

  CreateDirectory "$SMPROGRAMS\Zoop One"
  CreateShortCut "$SMPROGRAMS\Zoop One\Zoop One.lnk" "$INSTDIR\zoop-one.exe" "" "$INSTDIR\icon.ico"
  CreateShortCut "$DESKTOP\Zoop One.lnk" "$INSTDIR\zoop-one.exe" "" "$INSTDIR\icon.ico"

  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  WriteRegDWORD HKLM "${UNINSTALL_KEY}" "EstimatedSize" "$0"
SectionEnd

Section "Uninstall"
  DeleteRegKey HKLM "${UNINSTALL_KEY}"

  Delete "$SMPROGRAMS\Zoop One\Zoop One.lnk"
  Delete "$DESKTOP\Zoop One.lnk"
  RMDir "$SMPROGRAMS\Zoop One"

  RMDir /r "$INSTDIR"

  Delete "$APPDATA\Zoop\config.json"
  RMDir "$APPDATA\Zoop"
SectionEnd
