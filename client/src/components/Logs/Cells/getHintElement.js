import React, { useState } from 'react';
import PropTypes from 'prop-types';
import CustomTooltip from '../Tooltip/CustomTooltip';

const getHintElement = ({
    className,
    contentItemClass,
    columnClass,
    dataTip,
    xlinkHref,
    content,
    title,
    place,
    tooltipClass,
    trigger,
    overridePosition,
    scrollHide,
    renderContent,
}) => {
    const id = 'id';

    const [isHovered, hover] = useState(false);

    const openTooltip = () => hover(true);
    const closeTooltip = () => hover(false);

    return <div onMouseEnter={openTooltip}
                onMouseLeave={closeTooltip}>
        <div data-tip={dataTip}
             data-for={dataTip ? id : undefined}
             data-event={trigger}
        >
            {xlinkHref && <svg className={className}>
                <use xlinkHref={`#${xlinkHref}`} />
            </svg>}
        </div>
        {isHovered && dataTip
        && <CustomTooltip
            className={tooltipClass}
            id={id}
            columnClass={columnClass}
            contentItemClass={contentItemClass}
            title={title}
            place={place}
            content={content}
            trigger={trigger}
            overridePosition={overridePosition}
            scrollHide={scrollHide}
            renderContent={renderContent}
        />}
    </div>;
};

getHintElement.propTypes = {
    className: PropTypes.string,
    contentItemClass: PropTypes.string,
    columnClass: PropTypes.string,
    tooltipClass: PropTypes.string,
    title: PropTypes.string,
    place: PropTypes.string,
    dataTip: PropTypes.string,
    xlinkHref: PropTypes.string,
    overridePosition: PropTypes.func,
    scrollHide: PropTypes.bool,
    trigger: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.arrayOf(PropTypes.string),
    ]),
    content: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.array,
    ]),
    renderContent: PropTypes.arrayOf(PropTypes.element),
};

export default getHintElement;
