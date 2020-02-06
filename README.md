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

You could either clone this branch or apply the patch to your working branch

* Clone the repository

```
git clone https://github.com/iganeshk/AdGuardHome
cd AdGuardHome
```

OR

* Apply the patchfile

```
wget https://raw.githubusercontent.com/iganeshk/AdGuardHome/dev-assets/dark-mustard-theme.patch
git checkout master/your-branch
git apply dark-mustard-theme.patch
```

* Build the project

```
# comment out target builds which aren't required in the release.sh script
./release.sh
```

* Deploy the binary

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