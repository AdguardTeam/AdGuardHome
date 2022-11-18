import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';
import ReactModal from 'react-modal';
import { useDispatch } from 'react-redux';
import Form from './Form';
import { toggleLeaseModal } from '../../../../actions';

const Modal = ({
    isModalOpen,
    handleSubmit,
    processingAdding,
    cidr,
    rangeStart,
    rangeEnd,
    gatewayIp,
}) => {
    const dispatch = useDispatch();

    const toggleModal = () => dispatch(toggleLeaseModal());

    return (
        <ReactModal
            className="Modal__Bootstrap modal-dialog modal-dialog-centered modal-dialog--clients"
            closeTimeoutMS={0}
            isOpen={isModalOpen}
            onRequestClose={toggleModal}
        >
            <div className="modal-content">
                <div className="modal-header">
                    <h4 className="modal-title">
                        <Trans>dhcp_new_static_lease</Trans>
                    </h4>
                    <button type="button" className="close" onClick={toggleModal}>
                        <span className="sr-only">Close</span>
                    </button>
                </div>
                <Form
                    initialValues={{
                        mac: '',
                        ip: '',
                        hostname: '',
                        cidr,
                        rangeStart,
                        rangeEnd,
                        gatewayIp,
                    }}
                    onSubmit={handleSubmit}
                    processingAdding={processingAdding}
                    cidr={cidr}
                    rangeStart={rangeStart}
                    rangeEnd={rangeEnd}
                />
            </div>
        </ReactModal>
    );
};

Modal.propTypes = {
    isModalOpen: PropTypes.bool.isRequired,
    handleSubmit: PropTypes.func.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    cidr: PropTypes.string.isRequired,
    rangeStart: PropTypes.string,
    rangeEnd: PropTypes.string,
    gatewayIp: PropTypes.string,
};

export default withTranslation()(Modal);
