# BlobBrowser

A lightweight web server to list the contents of an Azure blob container with download links.

## Running Locally

1. Obtain the code and `cd blobbrowser/`.
1. Build the binary with `go build`.
1. Generate a SAS token URL in Azure and set it as an env var with something like `export BLOB_SAS_URL="https://mystorageaccount.blob.core.windows.net/mycontainer?sp=x&sig=y..."`.
1. Set [environment variables](#environment-variables). In particular, you'll want to use
1. Run with `./browser`.
1. Open a browser and go to `localhost:80`.

## Running on Azure

1. If you don't have a container registry already, create one. You can create an Azure Container Registry with `az acr create`.
1. Login to it with `az acr login --name myregistry`. See [Push and Pull docs](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-get-started-docker-cli?tabs=azure-cli).
1. Build an image with `docker build`. If using an Azure ACR, the command will be something like `docker build --platform="linux/amd64" --tag "myregistry.azurecr.io/images/blobbrowser" .`.
1. Push the image with `docker push`.
1. Set up an Azure App Service that runs the pushed image.
1. Set [environment variables](#environment-variables).
1. Give the app service a system-managed identity and give that identity some RBAC permissions over the storage account. Note, exact permissions still to be determined but `Storage Blob Data Contributor` is likely sufficient.

## Development

1. Install pre-commit hooks with `pre-commit install --install-hooks`.
1. Limit line length with `golines -w *.go`.
1. Format with `go fmt -w *.go`

## Environment Variables

1. *Optional* `USE_DEFAULT_CREDENTIAL="true"` will try several Azure authentication methods, such as CLI, VSCode and managed identity. Not setting this or setting it to any other value will use managed identity authentication. You will need to use this option when running locally.
1. *Mandatory* `AZURE_STORAGE_ACCOUNT_NAME="mystorageaccount"` will set the name of the Azure storage account.
1. *Mandatory* `AZURE_CONTAINER_NAME="testcontainer1"` will set the name of the Azure storage account container.
1. *Mandatory* `BLOBBROWSER_SECRET="cYdPWwBiUPm9pEcYdPWwBiUPm9pE"` is a password, which must be hashed with bcrypt. It will be used by users to access the `/list` page.
