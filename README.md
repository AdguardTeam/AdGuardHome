<p align="center">
  <img src="https://raw.githubusercontent.com/iganeshk/AdGuardHome/dev-assets/AdguardHome_DarkMustard_headshot.png" width="800px" alt="AdGuard Home Dark Mustard" />
</p>
<h3 align="center">Dark Mustard</h3>
<p align="center">
  Deep orange influenced dark theme for Adguard home.
</p>

<p align="center">
    <a href="https://github.com/AdguardTeam/AdGuardHome">AdguardTeam/AdGuardHome</a>
</p>

<br />

## Installation

Follow any of the below method and then deploy the binary

* #### Method: Docker ([iganesh/adguardhome/tags](https://hub.docker.com/r/iganesh/adguardhome/tags))

    ```
    # only edge builds at the moment
    docker pull iganesh/adguardhome:latest
    docker run --name adguardhome \
    -v /my/own/workdir:/opt/adguardhome/work \
    -v /my/own/confdir:/opt/adguardhome/conf \
    -p 53:53/tcp \
    -p 53:53/udp \
    -p 67:67/udp \
    -p 68:68/tcp \
    -p 68:68/udp \
    -p 80:80/tcp \
    -p 443:443/tcp \
    -p 853:853/tcp \
    -p 3000:3000/tcp \
    -d iganesh/adguardhome
    ```

* #### Method: Cloning the Repository

    ```
    git clone https://github.com/iganeshk/AdGuardHome
    cd AdGuardHome
    # comment out target builds which aren't required in the release.sh script
    ./release.sh
    ```

* #### Method: Applying the patch

    ```
    wget https://raw.githubusercontent.com/iganeshk/AdGuardHome/dev-assets/dark-mustard-theme.patch
    git checkout master/your-branch
    git apply dark-mustard-theme.patch
    # comment out target builds which aren't required in the release.sh script
    ./release.sh
    ```

* #### Method: Downloading the latest release

    [https://github.com/iganeshk/AdGuardHome/releases](https://github.com/iganeshk/AdGuardHome/releases)


* #### Deploy the binary (if built from source/downloaded latest release)

    ```
    # stop the Adguard Service
    service AdGuardHome stop

    cd /path/to/AdguardHome
    # copy/wget the target build
    # extract the newly built project binary (use the appropriate tarbet build)
    tar -xvf AdGuardHome_linux_amd64.tar.gz --strip-components=1 -C .

    # run the Adguard Service again
    service AdguardHome start

    # savor the darkness across the clients
    ```

## Screenshots

<p align="center">
    <img src="https://raw.githubusercontent.com/iganeshk/AdGuardHome/dev-assets/screenshot-dashboard.png" width="800px" alt="Screenshot-Dashboard" />
</p>
<p align="center">
    <img src="https://raw.githubusercontent.com/iganeshk/AdGuardHome/dev-assets/screenshot-settings.png" width="800px" alt="Screenshot-Settings" />
</p>
<p align="center">
    <img src="https://raw.githubusercontent.com/iganeshk/AdGuardHome/dev-assets/screenshot-filters.png" width="800px" alt="Screenshot-Filters" />
</p>
<p align="center">
    <img src="https://raw.githubusercontent.com/iganeshk/AdGuardHome/dev-assets/screenshot-query.png" width="800px" alt="Screenshot-Query-Log" />
</p>

## Feedback

* Create an issue if I missed any elements or you'd like them to get it patched.