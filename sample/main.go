package main

import "github.com/canhlinh/idg"

func main() {
	file := idg.NewFile("https://lh3.googleusercontent.com/GV1ZJoRpPxchfAarI96FQYTn6dFnEMSl7awVUNGt-54XR_8VxeZmnD_HjlGiQa39tgn5vUrI1Yr6aeI1lg=m22?ipbits=0", nil, nil)
	if _, err := idg.DownloadSingleFile(file, "./", 32); err != nil {
		panic(err)
	}
}
