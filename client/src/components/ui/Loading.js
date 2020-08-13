import React from 'react';
import PropTypes from 'prop-types';
import classNames from 'classnames';
import './Loading.css';

const Loading = ({ className }) => (
    <div className={classNames('loading', className)} />
);

Loading.propTypes = {
    className: PropTypes.string,
};

export default Loading;
