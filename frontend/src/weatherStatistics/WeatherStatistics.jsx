import { Button, TextField } from '@material-ui/core';
import useStyles from './styles';

export default function WeatherStatistics() {
  const classes = useStyles();

  return (
    <>            
      <main className={classes.main}>
        <div className={classes.input_container}> 
          <div className={classes.paper}>
            <form
              noValidate
              className={classes.form}
            >
              <TextField
                variant="outlined"
                margin="normal"
                required
                fullWidth
                id="City"
                label="City"
                name="City"
                autoFocus
              />
              <TextField
                variant="outlined"
                margin="normal"
                required
                fullWidth
                name="Years amount"
                label="Years amount"
                type="Years amount"
                id="Years amount"
              />
              <Button
                type="submit"
                fullWidth
                variant="contained"
                color="primary"
                className={classes.submit}
              >
                Get wind statistics
              </Button>
            </form>
          </div> 
        </div> 
      </main> 
    </>
  );
}
