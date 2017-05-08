import Overview from './Overview.js';
import Channels from './Channels.js';
import Legacy from './Legacy.js';
import Exit from './Exit.js';

var Router = ReactRouter.Router;
var Route = ReactRouter.Route;
var browserHistory = ReactRouter.browserHistory;

ReactDOM.render((
  <Router history={browserHistory}>
    <Route path='/exit' component={Exit}/>
    <Route path='/channels' component={Channels}/>
    <Route path='/legacy' component={Legacy}/>
    <Route path='*' component={Overview}/>
  </Router>
), document.getElementById('root'));