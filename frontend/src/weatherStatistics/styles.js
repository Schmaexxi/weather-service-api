import { makeStyles } from '@material-ui/core/styles';
import BGImage from './images/background.jpg';

const useStyles = makeStyles((theme) => ({
  main: {
    backgroundImage: `url(${BGImage})`,
    height: '100vh',
    backgroundSize: '100%'
  },
  input_container: {
    backgroundColor: '#FAD9C4',
    opacity: '0.7',
    height: '230px',
    width: '400px',
    borderRadius: '30px',
    display: 'flex',
    margin: 'auto',
  },
  form: {
    width: '50%',
    marginTop: theme.spacing(1),
  },
  paper: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
  },
  submit: {
    margin: theme.spacing(2, 0, 1),
  },
}));

export default useStyles;

