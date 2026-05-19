Output docs/assets/oci-hop-demo.gif

Require bash

Set Shell "bash"
Set FontSize 22
Set FontFamily "Menlo"
Set LetterSpacing 0
Set LineHeight 1.2
Set Width 1180
Set Height 680
Set Theme "TokyoNight"
Set Padding 18
Set WindowBar Colorful
Set BorderRadius 8
Set TypingSpeed 35ms
Set Framerate 30

Hide
Type "export PATH=./docs/demo/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"
Enter
Type "export PS1='$ '"
Enter
Type "clear"
Enter
Show

Sleep 700ms
Type "hop explain my-vps-01"
Enter
Sleep 1400ms
Type "hop my-vps-01"
Enter
Sleep 1200ms
Type "ssh my-vps-01"
Enter
Sleep 2200ms
