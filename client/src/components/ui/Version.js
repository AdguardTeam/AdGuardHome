import React from 'react';
import { Trans, useTranslation } from 'react-i18next';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import { getVersion } from '../../actions';
import './Version.css';

const Version = () => {
    const dispatch = useDispatch();
    const { t } = useTranslation();
    const {
        dnsVersion,
        processingVersion,
        checkUpdateFlag,
    } = useSelector((state) => state?.dashboard ?? {}, shallowEqual);

    const {
        dnsVersion: installDnsVersion,
    } = useSelector((state) => state?.install ?? {}, shallowEqual);

    const version = dnsVersion || installDnsVersion;

    const onClick = () => {
        dispatch(getVersion(true));
    };

    return (
        <div className="version">
            <div className="version__text">
                {version && (
                    <>
                        <Trans>version</Trans>:&nbsp;
                        <span className="version__value" title={version}>{version}</span>
                    </>
                )}
                {checkUpdateFlag && <button
                    type="button"
                    className="btn btn-icon btn-icon-sm btn-outline-primary btn-sm ml-2"
                    onClick={onClick}
                    disabled={processingVersion}
                    title={t('check_updates_now')}
                >
                    <svg className="icons icon12">
                        <use xlinkHref="#refresh" />
                    </svg>
                </button>}
            </div>
        </div>
    );
};

export default Version;
