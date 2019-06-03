import { connect } from 'react-redux';
import { addErrorToast } from '../actions';
import { addClient, updateClient, deleteClient, toggleClientModal } from '../actions/clients';
import Clients from '../components/Settings/Clients';

const mapStateToProps = (state) => {
    const { dashboard, clients } = state;
    const props = {
        dashboard,
        clients,
    };
    return props;
};

const mapDispatchToProps = {
    addErrorToast,
    addClient,
    updateClient,
    deleteClient,
    toggleClientModal,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Clients);
