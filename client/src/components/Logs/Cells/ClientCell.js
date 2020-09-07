import React, { useState } from 'react';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import { nanoid } from 'nanoid';
import classNames from 'classnames';
import { useTranslation } from 'react-i18next';
import propTypes from 'prop-types';
import { checkFiltered, getBlockingClientName } from '../../../helpers/helpers';
import { BLOCK_ACTIONS } from '../../../helpers/constants';
import { toggleBlocking, toggleBlockingForClient } from '../../../actions';
import IconTooltip from './IconTooltip';
import { renderFormattedClientCell } from '../../../helpers/renderFormattedClientCell';
import { toggleClientBlock } from '../../../actions/access';
import { getBlockClientInfo } from './helpers';

const ClientCell = ({
    client,
    domain,
    info,
    info: { name, whois_info },
    reason,
}) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const autoClients = useSelector((state) => state.dashboard.autoClients, shallowEqual);
    const processingRules = useSelector((state) => state.filtering.processingRules);
    const isDetailed = useSelector((state) => state.queryLogs.isDetailed);
    const [isOptionsOpened, setOptionsOpened] = useState(false);

    const disallowed_clients = useSelector(
        (state) => state.access.disallowed_clients,
        shallowEqual,
    );

    const autoClient = autoClients.find((autoClient) => autoClient.name === client);
    const source = autoClient?.source;
    const whoisAvailable = whois_info && Object.keys(whois_info).length > 0;

    const id = nanoid();

    const data = {
        address: client,
        name,
        country: whois_info?.country,
        city: whois_info?.city,
        network: whois_info?.orgname,
        source_label: source,
    };

    const processedData = Object.entries(data);

    const isFiltered = checkFiltered(reason);

    const nameClass = classNames('w-90 o-hidden d-flex flex-column', {
        'mt-2': isDetailed && !name && !whoisAvailable,
        'white-space--nowrap': isDetailed,
    });

    const hintClass = classNames('icons mr-4 icon--24 icon--lightgray', {
        'my-3': isDetailed,
    });

    const renderBlockingButton = (isFiltered, domain) => {
        const buttonType = isFiltered ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;
        const clients = useSelector((state) => state.dashboard.clients);

        const {
            confirmMessage,
            buttonKey: blockingClientKey,
            type,
        } = getBlockClientInfo(client, disallowed_clients);

        const blockingForClientKey = isFiltered ? 'unblock_for_this_client_only' : 'block_for_this_client_only';
        const clientNameBlockingFor = getBlockingClientName(clients, client);

        const BUTTON_OPTIONS_TO_ACTION_MAP = {
            [blockingForClientKey]: () => {
                dispatch(toggleBlockingForClient(buttonType, domain, clientNameBlockingFor));
            },
            [blockingClientKey]: () => {
                const message = `${type === BLOCK_ACTIONS.BLOCK ? t('adg_will_drop_dns_queries') : ''} ${t(confirmMessage, { ip: client })}`;
                if (window.confirm(message)) {
                    dispatch(toggleClientBlock(type, client));
                }
            },
        };

        const onClick = () => dispatch(toggleBlocking(buttonType, domain));

        const getOptions = (optionToActionMap) => {
            const options = Object.entries(optionToActionMap);
            if (options.length === 0) {
                return null;
            }
            return <>{options
                .map(([name, onClick]) => <div
                    key={name}
                    className="button-action--arrow-option px-4 py-2"
                    onClick={onClick}
            >{t(name)}
            </div>)}</>;
        };

        const content = getOptions(BUTTON_OPTIONS_TO_ACTION_MAP);

        const buttonClass = classNames('button-action button-action--main', {
            'button-action--unblock': isFiltered,
            'button-action--with-options': content,
            'button-action--active': isOptionsOpened,
        });

        const buttonArrowClass = classNames('button-action button-action--arrow', {
            'button-action--unblock': isFiltered,
            'button-action--active': isOptionsOpened,
        });

        const containerClass = classNames('button-action__container', {
            'button-action__container--detailed': isDetailed,
        });

        return <div className={containerClass}>
            <button type="button"
                    className={buttonClass}
                    onClick={onClick}
                    disabled={processingRules}
            >
                {t(buttonType)}
            </button>
            {content && <button className={buttonArrowClass} disabled={processingRules}>
                <IconTooltip
                        className='h-100'
                        tooltipClass='button-action--arrow-option-container'
                        xlinkHref='chevron-down'
                        triggerClass='button-action--icon'
                        content={content} placement="bottom-end" trigger="click"
                        onVisibilityChange={setOptionsOpened}
                />
            </button>}
        </div>;
    };

    return <div className="o-hidden h-100 logs__cell logs__cell--client" role="gridcell">
        <IconTooltip className={hintClass} columnClass='grid grid--limited' tooltipClass='px-5 pb-5 pt-4 mw-75'
                     xlinkHref='question' contentItemClass="contentItemClass" title="client_details"
                     content={processedData} placement="bottom" />
        <div className={nameClass}>
            <div data-tip={true} data-for={id}>
                {renderFormattedClientCell(client, info, isDetailed, true)}
            </div>
            {isDetailed && name && !whoisAvailable
            && <div className="detailed-info d-none d-sm-block logs__text"
                    title={name}>{name}</div>}
        </div>
        {renderBlockingButton(isFiltered, domain)}
    </div>;
};

ClientCell.propTypes = {
    client: propTypes.string.isRequired,
    domain: propTypes.string.isRequired,
    info: propTypes.oneOfType([
        propTypes.string,
        propTypes.shape({
            name: propTypes.string.isRequired,
            whois_info: propTypes.shape({
                country: propTypes.string,
                city: propTypes.string,
                orgname: propTypes.string,
            }),
        }),
    ]),
    reason: propTypes.string.isRequired,
};

export default ClientCell;
