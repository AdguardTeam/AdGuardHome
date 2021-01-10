import React from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import { useTranslation } from 'react-i18next';
import './Loading.css';

const Loading = ({ className, text }) => {
    const { t } = useTranslation();
    return <div className={classNames('loading', className)}>{t(text)}</div>;
};

Loading.propTypes = {
    className: PropTypes.string,
    text: PropTypes.string,
};

export default Loading;
