Output notation-demo.mp4

Require echo
Require nomad

Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600


Set TypingSpeed 100ms

Type `echo "Let's build notation's quickstart demo image"` Enter
Type "docker build -t localhost:5001/net-monitor:v1 https://github.com/wabbit-networks/net-monitor.git\#main" Sleep 500ms  Enter
Sleep 5s
Type "docker push localhost:5001/net-monitor:v1" Sleep 500ms  Enter
Sleep 2s

Type "clear" Enter
Set TypingSpeed 150ms
Type `echo "Now export the image digest and store it in a variable"` Enter
Enter
Type `export IMAGE=$(docker inspect --format='{{index .RepoDigests 0}}' localhost:5001/net-monitor:v1)` Sleep 500ms  Enter
Sleep 500ms
Hide

Type "export NOMAD_ADDR='http://localhost:6464'" Enter

Show
Enter
Type `echo "Let's try to deploy it to Nomad"`  Enter
Sleep 1s

Type `nomad run -var "image=${IMAGE}" demo.nomad` Sleep 200ms  Enter Enter
Sleep 5s
Type `echo "It failed because the image wasn't signed, let's sign and redeploy it"`  Enter
Sleep 2s
Type `notation sign $IMAGE` Sleep 500ms  Enter
Sleep 3s
Type `nomad run -var "image=${IMAGE}" demo.nomad` Enter Sleep 5s

Type `echo "yey, it worked!"` Enter Sleep 5s
