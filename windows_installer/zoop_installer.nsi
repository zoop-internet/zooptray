; NSIS installer for Zoop One
!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "unknown"
!endif

Name "Zoop One"
OutFile "zoop-one-installer-${VERSION}.exe"
InstallDir "$PROGRAMFILES64\\Zoop\\One"
RequestExecutionLevel admin

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "EULA.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Languages
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  ; Copy app files from staging folder
  File "staging\\zoop-one.exe"

  ; Ensure AppData folder and write default config (with EULA accepted)
  SetOutPath "$APPDATA\\Zoop"
  File "staging\\config.json"

  ; Create shortcuts
  CreateDirectory "$SMPROGRAMS\\Zoop One"
  CreateShortcut "$SMPROGRAMS\\Zoop One\\Zoop One.lnk" "$INSTDIR\\zoop-one.exe"
  CreateShortcut "$DESKTOP\\Zoop One.lnk" "$INSTDIR\\zoop-one.exe"
SectionEnd

Section "Uninstall"
  Delete "$SMPROGRAMS\\Zoop One\\Zoop One.lnk"
  Delete "$DESKTOP\\Zoop One.lnk"
  RMDir "$SMPROGRAMS\\Zoop One"
  RMDir /r "$INSTDIR"
  Delete "$APPDATA\\Zoop\\config.json"
  RMDir "$APPDATA\\Zoop"
SectionEnd
