# Configuring Netcup Credentials

To use the Netcup DNS provider, you must configure your credentials. The application requires three pieces of information to authenticate with the Netcup API: **Customer Number**, an **API Key**, and the corresponding **API Password**.

Set `DNS_PROVIDER=netcup_ccp` to select this provider (it is the default). If you are storing credentials in `.env.local`, you can keep them in that file and rely on the application to load them automatically.

You can find your Customer Number and generate an API Key and Password in the "Master Data" (`Stammdaten`) section of your Netcup Customer Control Panel (CCP).

There are two ways to provide these credentials, with the following order of precedence:

1.  **Environment Variables (Highest Priority)**
2.  **Credentials File**

## 1. Environment Variables

Set the following environment variables with your credentials. If all three are set, the credentials file will not be read.

- `NETCUP_CUSTOMER_NUMBER`: Your Netcup customer number.
- `NETCUP_API_KEY`: The API key you generated in the CCP.
- `NETCUP_API_PASSWORD`: The password associated with your API key.

### Example

```sh
export NETCUP_CUSTOMER_NUMBER="12345"
export NETCUP_API_KEY="xxxxxxxxxxxxxxxxxxxxxxxx"
export NETCUP_API_PASSWORD="your-secret-api-password"
```

## 2. Credentials File

If any of the environment variables are missing, the application will look for a credentials file at `~/.homeddns/netcup_credentials`.

Create this file and add your credentials in the following format:

```ini
# ~/.homeddns/netcup_credentials

customer_number = 12345
api_key = xxxxxxxxxxxxxxxxxxxxxxxx
api_password = your-secret-api-password
```

The application will use the values from this file to fill in any credentials that were not provided by environment variables.
