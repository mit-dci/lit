import lc from './LitClient';
import Q from 'q';

class MetaCmds extends Reflux.Component {
  stop () {
    lc.send('LitRPC.Stop')
    .then(res => {
      window.location = '/exit';
    })
    .fail(err => {
      console.error(err);
    });
  }
  render () {
    return (
      <div>
        <h2>Meta Commands</h2>
        <button onClick={this.stop}>stop</button>
      </div>
    );
  }
}

export default MetaCmds;