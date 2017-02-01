[![Coverage](http://esheavyindustries.com:8080/display?repo=hoptocopter_git)](http://esheavyindustries.com:8080/display?repo=hoptocopter_git)



# hoptocopter


Getting a test coverage badge into your README without a 3rd party SaaS tool shouldn't be that hard but sadly it is.

hoptocopter will let you POST a coverage file output from `go test -coverprofile=coverage.out` and spit out the badge for you thanks to [shields.io](https://shields.io).

But wait, didn't I just say you don't have to use a 3rd party tool? I did. [shields.io](https://shields.io) is nice enough to open source their stuff and beevelop was nice 
enough to bundle it up in a [Docker image](https://github.com/beevelop/docker-shields). With the magic of Docker compose you can run all of this together on your own instance.

Run the app anywhere you like. No need to integrate with any other 3rd party for auth or repo access.

Also note that most of this code was 100% lifted from the stdlib of Go, I just wrapped an HTTP endpoint around it.


# Install

`git clone https://git.esheavyindustries.com/esell/hoptocopter.git`
`cd hoptocopter`
`docker-compose up`


Now all you need to do is POST your coverage output to hoptocopter:
`curl -XPOST 'http://myserver.com:8080/upload?repo=my-cool-app' -F "file=@coverage.out"`

And when you want the badge? Just send a GET hoptocopter's way:
`curl -XPOST 'http://localhost:8080/display?repo=deb-simple'"`
