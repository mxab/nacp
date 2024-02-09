Output otel-demo.mp4

Require echo
Require nomad
Require vim
Require bat


Set Shell "bash"
Set FontSize 16
Set Width 1200
Set Height 600


Set TypingSpeed 100ms

Type `echo "Let's run a simple nomad job"` Enter
Type `vim demo.nomad.hcl` Enter
Set TypingSpeed 50ms
Type i

Type `job "example" {` Enter
Type ` group "demo" {` Enter
Type `  task "app" {` Enter
Type `   driver = "docker"` Enter
Type `   config {` Enter
Type `    image = "busybox:1.36.1"` Enter
Type `     args  = ["/bin/sh", "-c", "env && sleep 1h"]` Enter
Type `   }` Enter
Type `  }` Enter
Type ` }` Enter
Type `}` Enter
Escape
Type `:wq` Enter
Set TypingSpeed 100ms
Sleep 2s


Type `nomad run -detach demo.nomad.hcl` Enter
Sleep 5s
Type `echo "Let's check the env vars of our job"` Enter
Type `nomad exec -job example sh -c 'env | grep OTEL_'` Enter
Sleep 1s
Type `echo "No 'OTEL_' env vars, let's update the job to enable otel"` Enter
Type `vim demo.nomad.hcl` Enter
Sleep 1s
Down Down Down Down
Type i
Enter
Up
Sleep 500ms
Type `   meta { otel = "true" }`
Sleep 500ms
Escape
Type `:wq` Enter
Sleep 1s
Type `echo "Let's run the updated job"` Enter
Type `nomad run -detach demo.nomad.hcl` Enter
Sleep 6s
Type `echo "Let's check the env vars of our job again"` Enter
Type `nomad exec -job example sh -c 'env | grep OTEL_'` Enter
Sleep 1s

Type `echo "yey, it worked!"` Enter Sleep 5s
