# BlobBrowser

A lightweight web server to list the contents of an Azure blob container with download links.

## Running Locally

1. Obtain the code and `cd blobbrowser/`.
1. Build the binary with `go build`.
1. Generate a SAS token URL in Azure and set it as an env var with something like `export BLOB_SAS_URL="https://mystorageaccount.blob.core.windows.net/mycontainer?sp=x&sig=y..."`.
1. Run with `./browser`.
1. Open a browser and go to `localhost:80`.

## Running on Azure

1. If you don't have a container registry already, create one. You can create an Azure Container Registry with `az acr create`.
1. Login to it with `az acr login --name myregistry`. See [Push and Pull docs](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-get-started-docker-cli?tabs=azure-cli).
1. Build an image with `docker build`. If using an Azure ACR, the command will be something like `docker build --platform="linux/amd64" --tag "myregistry.azurecr.io/images/blobbrowser" .`.
1. Push the image with `docker push`.
1. Set up an Azure App Service that runs the pushed image.
1. Generate a SAS token URL in Azure and set it as an env var called `BLOB_SAS_URL`.

## Developement

1. Install pre-commit hooks with `pre-commit install --install-hooks`.
1. Limit line length with `golines -w *.go`.
1. Format with `go fmt -w *.go`
