import React from 'react';
import { Trans, withNamespaces } from 'react-i18next';

import Tabs from '../ui/Tabs';
import Icons from '../ui/Icons';

const Guide = () => (
    <div>
        <Icons />
        <Tabs>
            <div label="Router">
                <div className="tab__title">
                    <Trans>install_devices_router</Trans>
                </div>
                <div className="tab__text">
                    <p><Trans>install_devices_router_desc</Trans></p>
                    <ol>
                        <li><Trans>install_devices_router_list_1</Trans></li>
                        <li><Trans>install_devices_router_list_2</Trans></li>
                        <li><Trans>install_devices_router_list_3</Trans></li>
                    </ol>
                </div>
            </div>
            <div label="Windows">
                <div className="tab__title">
                    Windows
                </div>
                <div className="tab__text">
                    <ol>
                        <li><Trans>install_devices_windows_list_1</Trans></li>
                        <li><Trans>install_devices_windows_list_2</Trans></li>
                        <li><Trans>install_devices_windows_list_3</Trans></li>
                        <li><Trans>install_devices_windows_list_4</Trans></li>
                        <li><Trans>install_devices_windows_list_5</Trans></li>
                        <li><Trans>install_devices_windows_list_6</Trans></li>
                    </ol>
                </div>
            </div>
            <div label="macOS">
                <div className="tab__title">
                    macOS
                </div>
                <div className="tab__text">
                    <ol>
                        <li><Trans>install_devices_macos_list_1</Trans></li>
                        <li><Trans>install_devices_macos_list_2</Trans></li>
                        <li><Trans>install_devices_macos_list_3</Trans></li>
                        <li><Trans>install_devices_macos_list_4</Trans></li>
                    </ol>
                </div>
            </div>
            <div label="Android">
                <div className="tab__title">
                    Android
                </div>
                <div className="tab__text">
                    <ol>
                        <li><Trans>install_devices_android_list_1</Trans></li>
                        <li><Trans>install_devices_android_list_2</Trans></li>
                        <li><Trans>install_devices_android_list_3</Trans></li>
                        <li><Trans>install_devices_android_list_4</Trans></li>
                        <li><Trans>install_devices_android_list_5</Trans></li>
                    </ol>
                </div>
            </div>
            <div label="iOS">
                <div className="tab__title">
                    iOS
                </div>
                <div className="tab__text">
                    <ol>
                        <li><Trans>install_devices_ios_list_1</Trans></li>
                        <li><Trans>install_devices_ios_list_2</Trans></li>
                        <li><Trans>install_devices_ios_list_3</Trans></li>
                        <li><Trans>install_devices_ios_list_4</Trans></li>
                    </ol>
                </div>
            </div>
        </Tabs>
    </div>
);

export default withNamespaces()(Guide);
