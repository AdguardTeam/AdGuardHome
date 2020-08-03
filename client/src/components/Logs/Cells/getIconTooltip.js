import React from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';
import classNames from 'classnames';
import { processContent } from '../../../helpers/helpers';
import Tooltip from '../../ui/Tooltip';
import 'react-popper-tooltip/dist/styles.css';
import './IconTooltip.css';

const getIconTooltip = ({
    className,
    contentItemClass,
    columnClass,
    canShowTooltip = true,
    xlinkHref,
    title,
    placement,
    tooltipClass,
    content,
    renderContent = content ? React.Children.map(
        processContent(content),
        (item, idx) => <div key={idx} className={contentItemClass}>
            <Trans>{item || '—'}</Trans>
        </div>,
    ) : null,
}) => {
    const tooltipContent = <>
        {title
        && <div className="pb-4 h-25 grid-content font-weight-bold"><Trans>{title}</Trans></div>}
        <div className={classNames(columnClass)}>{renderContent}</div>
    </>;

    const tooltipClassName = classNames('tooltip-custom__container', tooltipClass, { 'd-none': !canShowTooltip });

    return <Tooltip
        className={tooltipClassName}
        content={tooltipContent}
        placement={placement}
    >
        {xlinkHref && <svg className={className}>
            <use xlinkHref={`#${xlinkHref}`} />
        </svg>}
    </Tooltip>;
};

getIconTooltip.propTypes = {
    className: PropTypes.string,
    contentItemClass: PropTypes.string,
    columnClass: PropTypes.string,
    tooltipClass: PropTypes.string,
    title: PropTypes.string,
    placement: PropTypes.string,
    canShowTooltip: PropTypes.string,
    xlinkHref: PropTypes.string,
    content: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.array,
    ]),
    renderContent: PropTypes.arrayOf(PropTypes.element),
};

export default getIconTooltip;
