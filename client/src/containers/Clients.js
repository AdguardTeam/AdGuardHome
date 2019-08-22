import { connect } from 'react-redux';
import { getClients } from '../actions';
import { getStats } from '../actions/stats';
import { addClient, updateClient, deleteClient, toggleClientModal } from '../actions/clients';
import Clients from '../components/Settings/Clients';

const mapStateToProps = (state) => {
    const { dashboard, clients, stats } = state;
    const props = {
        dashboard,
        clients,
        stats,
    };
    return props;
};

const mapDispatchToProps = {
    getClients,
    getStats,
    addClient,
    updateClient,
    deleteClient,
    toggleClientModal,
};

export default connect(
    mapStateToProps,
    mapDispatchToProps,
)(Clients);
