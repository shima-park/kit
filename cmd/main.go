package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"strings"

	"ezrpro.com/micro/kit/pkg/cst"
	gen "ezrpro.com/micro/kit/pkg/generator"
	"ezrpro.com/micro/kit/pkg/generator/endpoint"
	"ezrpro.com/micro/kit/pkg/generator/grpc"
)

var (
	sourceFile   = "./addservice/addservice.go"
	pbSourceFile = "./test.pb.go"
	useTest      = flag.Bool("t", true, "Use test func")
	grpcToGo     = flag.Bool("g", false, "Exec protoc -I ./ ./test.proto --go_out=plugins=grpc:./")
	genProto     = flag.Bool("p", false, "Generate protobuf file by source file")
	genEndpoint  = flag.Bool("e", false, "Generate endpoint go source file by source file")
)

func main() {
	flag.Parse()

	if *useTest {
		filename := sourceFile
		if *grpcToGo {
			filename = pbSourceFile
		}
		test(filename)
	} else {
		testASTPrint()
	}

}

func testASTPrint() {
	fset := token.NewFileSet() // positions are relative to fset

	// Parse the file containing this very example
	// but stop after processing the imports.
	f, err := parser.ParseFile(fset, sourceFile, nil, 0)
	if err != nil {
		panic(err)
	}

	ast.Print(fset, f)
}

func test(sourceFilename string) {

	fset := token.NewFileSet() // positions are relative to fset

	// Parse the file containing this very example
	// but stop after processing the imports.
	f, err := parser.ParseFile(fset, sourceFilename, nil, 0)
	if err != nil {
		panic(err)
	}

	// Remove the first variable declaration from the list of declarations.
	a := cst.NewConcreteSyntaxTree(
		fset,
		f,
	)
	a.Parse()

	//fmt.Println(a.String())

	if *genProto {
		filename := "./test.proto"
		file := CreateFile(filename)
		defer file.Close()
		gen := grpc.NewGRPCGenerator(a, grpc.WithWriter(file))
		err := gen.Generate()
		if err != nil {
			panic(err)
		}
	}

	if *grpcToGo {
		filename := "./test.proto"
		generateProtobufGo(filename)
	}

	if *genEndpoint {
		filename := "./addendpoint.go"
		file := CreateFile(filename)
		defer file.Close()
		gen := endpoint.NewEndpointGenerator(a,
			endpoint.WithWriter(file),
			endpoint.WithTemplateConfig(
				gen.NewTemplateConfig(a),
			),
		)
		err := gen.Generate()
		if err != nil {
			panic(err)
		}
		formatGoSourceFile(filename)
	}

}

func CreateFile(filename string) io.WriteCloser {
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			file, err := os.Create(filename)
			if err != nil {
				panic(err)
			}
			return file
		}
		panic(err)
	}

	err = os.Remove(filename)
	if err != nil {
		fmt.Println(err)
	}

	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	return file
}

func generateProtobufGo(protoPath string) {
	genPbPath := "./"
	args := []string{
		"-I", genPbPath,
		protoPath,
		"--go_out=plugins=grpc:" + genPbPath,
	}
	//protoc -I ./ --go_out=plugins=grpc:./ ./test.proto
	cmd := exec.Command("protoc", args...)

	fmt.Println("protoc", strings.Join(args, " "))

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}

func formatGoSourceFile(filePath string) {
	args := []string{
		"-w", filePath,
	}
	cmd := exec.Command("gofmt", args...)

	fmt.Println("gofmt", strings.Join(args, " "))

	err := cmd.Run()
	if err != nil {
		panic(err)
	}
}
