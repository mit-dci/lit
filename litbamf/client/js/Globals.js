import bamflogo from './bamf.ascii.js';

class Navbar extends Reflux.Component {
  constructor (props) {
    super(props);
  }

  render () {
    return (
      <nav>
        <pre>{bamflogo}</pre>
        <ul>
          <li className={this.props.page == 'overview' ? 'checked' : ''}>
            <a href="/overview">Overview</a>
          </li>
          <li className={this.props.page == 'channels' ? 'checked' : ''}>
            <a href="/channels">Channels</a>
          </li>
          <li className={this.props.page == 'legacy' ? 'checked' : ''}>
            <a href="/legacy">Legacy</a>
          </li>
        </ul>
      </nav>
    );
  }
}

export {Navbar};