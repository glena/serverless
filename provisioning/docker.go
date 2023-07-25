package provisioning

import (
	"archive/tar"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func readFile(file string, path string, tw *tar.Writer) {
	fileReader, err := os.Open(path)
	if err != nil {
		log.Fatal(err, " :unable to open Dockerfile")
	}
	readFile, err := ioutil.ReadAll(fileReader)
	if err != nil {
		log.Fatal(err, " :unable to read dockerfile")
	}

	tarHeader := &tar.Header{
		Name: file,
		Size: int64(len(readFile)),
	}
	err = tw.WriteHeader(tarHeader)
	if err != nil {
		log.Fatal(err, " :unable to write tar header")
	}
	_, err = tw.Write(readFile)
	if err != nil {
		log.Fatal(err, " :unable to write tar body")
	}
}

func CreateDockerImage(name string) (err error) {
	ctx := context.Background()

	cli, err := client.NewEnvClient()

	if err != nil {
		log.Fatal(err, " :unable to init client")
		return err
	}

	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	defer tw.Close()

	dockerFile := "Dockerfile"

	readFile(dockerFile, "./docker/nodejs/Dockerfile", tw)
	readFile("server.js", "./docker/nodejs/server.js", tw)

	tarReader := bytes.NewReader(buf.Bytes())

	imageBuildResponse, err := cli.ImageBuild(
		ctx,
		tarReader,
		types.ImageBuildOptions{
			Tags:       []string{name},
			Context:    tarReader,
			Dockerfile: dockerFile,
			Remove:     true})

	if err != nil {
		log.Fatal(err, " :unable to build docker image")
		return err
	}
	defer imageBuildResponse.Body.Close()
	_, err = io.Copy(os.Stdout, imageBuildResponse.Body)
	if err != nil {
		log.Fatal(err, " :unable to read image build response")
		return err
	}

	return nil
}
