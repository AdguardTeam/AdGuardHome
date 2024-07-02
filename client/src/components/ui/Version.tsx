import React from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';

import { getVersion } from '../../actions';
import './Version.css';
import { RootState } from '../../initialState';

const Version = () => {
    const dispatch = useDispatch();
    const { t } = useTranslation();
    const dashboard = useSelector((state: RootState) => state.dashboard, shallowEqual);
    const install = useSelector((state: RootState) => state.install, shallowEqual);

    if (!dashboard && !install) {
        return null;
    }

    const version = dashboard?.dnsVersion || install?.dnsVersion;

    const onClick = () => {
        dispatch(getVersion(true));
    };

    return (
        <div className="version">
            <div className="version__text">
                {version && (
                    <>
                        <Trans>version</Trans>:&nbsp;
                        <span className="version__value" title={version}>
                            {version}
                        </span>
                    </>
                )}

                {dashboard?.checkUpdateFlag && (
                    <button
                        type="button"
                        className="btn btn-icon btn-icon-sm btn-outline-primary btn-sm ml-2"
                        onClick={onClick}
                        disabled={dashboard?.processingVersion}
                        title={t('check_updates_now')}>
                        <svg className="icons icon12">
                            <use xlinkHref="#refresh" />
                        </svg>
                    </button>
                )}
            </div>
        </div>
    );
};

export default Version;
