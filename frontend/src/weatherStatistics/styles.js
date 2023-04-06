import { makeStyles } from '@material-ui/core/styles';
import BGImage from './images/background.jpg';

const useStyles = makeStyles((theme) => ({
  main: {
    backgroundImage: `url(${BGImage})`,
    height: '100vh',
    backgroundSize: '100%'
  },
  container: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
  },
  input_container: {
    marginTop: '10px',
    backgroundColor: '#FAD9C4',
    opacity: '0.7',
    height: '300px',
    width: '600px',
    borderRadius: '30px',
    display: 'flex',
    margin: 'auto',
    flexDirection: 'column',
    alignItems: 'center',
  },
  form: {
    width: '70%',
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
  chart: {
    marginTop: theme.spacing(15),
    padding: '10px',
    backgroundColor: 'black',
    width: '900px',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    borderRadius: '20px',
    opacity: '0.9',
  },
  title: {
    color: 'red',
  },
  error:{
    color: 'red',
    fontSize: '18px',
    marginBottom: '10px',
    textAlign: 'center',
  },
}));

export default useStyles;

