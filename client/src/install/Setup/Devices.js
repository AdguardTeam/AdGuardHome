import React from 'react';
import { Trans } from 'react-i18next';

import Tabs from '../../components/ui/Tabs';
import Icons from '../../components/ui/Icons';
import Controls from './Controls';

const Devices = () => (
    <div className="setup__step">
        <div className="setup__group">
            <div className="setup__subtitle">
                <Trans>install_devices_title</Trans>
            </div>
            <p className="setup__desc">
                <Trans>install_devices_desc</Trans>
            </p>
            <Icons />
            <Tabs>
                <div label="Router">
                    <div className="tab__title">
                        <Trans>install_decices_router</Trans>
                    </div>
                    <div className="tab__text">
                        <Trans>install_decices_router_desc</Trans>
                        <ol>
                            <li>
                                <Trans>install_decices_router_list_1</Trans>
                            </li>
                            <li>
                                <Trans>install_decices_router_list_2</Trans>
                            </li>
                            <li>
                                <Trans>install_decices_router_list_3</Trans>
                            </li>
                        </ol>
                    </div>
                </div>
                <div label="Windows">
                    <div className="tab__title">
                        Windows
                    </div>
                    <div className="tab__text">Lorem ipsum dolor sit amet consectetur adipisicing elit. Deleniti sapiente magnam autem excepturi repellendus, voluptatem officia sint quas nulla maiores velit odit dolore commodi quia reprehenderit vero repudiandae adipisci aliquam.</div>
                </div>
                <div label="macOS">
                    <div className="tab__title">
                        macOS
                    </div>
                    <div className="tab__text">Lorem ipsum dolor sit amet consectetur adipisicing elit. Deleniti sapiente magnam autem excepturi repellendus, voluptatem officia sint quas nulla maiores velit odit dolore commodi quia reprehenderit vero repudiandae adipisci aliquam.</div>
                </div>
                <div label="Android">
                    <div className="tab__title">
                        Android
                    </div>
                    <div className="tab__text">Lorem ipsum dolor sit amet consectetur adipisicing elit. Deleniti sapiente magnam autem excepturi repellendus, voluptatem officia sint quas nulla maiores velit odit dolore commodi quia reprehenderit vero repudiandae adipisci aliquam.</div>
                </div>
                <div label="iOS">
                    <div className="tab__title">
                        iOS
                    </div>
                    <div className="tab__text">Lorem ipsum dolor sit amet consectetur adipisicing elit. Deleniti sapiente magnam autem excepturi repellendus, voluptatem officia sint quas nulla maiores velit odit dolore commodi quia reprehenderit vero repudiandae adipisci aliquam.</div>
                </div>
            </Tabs>
        </div>
        <Controls />
    </div>
);

export default Devices;
