<#
.NOTES

    Summary: Windows native build script. This poewrshell script was created to mimic the build functionalities provided 
             through "make" + Makefile on Linux for Windows build environment

             Usage Examples (run from repo root: dcos-check-runner):

                ".\scripts\make.ps1 all"
                ".\scripts\make.ps1 build"
                ".\scripts\make.ps1 test"
                ".\scripts\make.ps1 clean"
                ".\scripts\make.ps1 install"
#>

$CheckRunnerServiceFileName = "dcos-check-runner.exe"

Function makeTest
{
    & go get github.com/stretchr/testify
    powershell.exe -F './scripts/test.ps1'
}

Function makeInstall
{
    & go install
} 

Function makeBuild
{
    & go build
} 

Function makeClean
{
    if (Test-Path -Path "./$CheckRunnerServiceFileName") {
        del ./dcos-check-runner.exe
    }

    $gopath = (get-item env:"GOPATH").Value + "\bin"
    if (Test-Path -Path "$gopath/$CheckRunnerServiceFileName") {
        del $gopath/$CheckRunnerServiceFileName
        Write-Output "found  $gopath/$CheckRunnerServiceFileName and deleted"
    }
} 

Function DoMake 
{
 Param(
    [string]$Target) 

    Write-Output "make $Target"
    switch ( $Target )
    {
        test
        {
            makeTest
            break
        }
        build
        {
            makeBuild
            break
        }
        install
        {
            makeInstall
            break
        }
        clean
        {
            makeClean
            break
        }
        all
        {
            makeTest
            makeInstall
            break
        }
    }
} 


if (-NOT ($args.Count -eq 1))
{
    Write-Output ("Usage: Build {all, test, build, install, or clean}")
    exit -1
}

DoMake -Target $args
