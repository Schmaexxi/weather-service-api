import { Button, TextField } from '@material-ui/core';
import useStyles from './styles';
import { useState } from 'react';

export default function WeatherStatistics() {
  const classes = useStyles();

  const [city, setCity] = useState('');
  const [years, setYears] = useState(0);

  const [errorMessage, setErrorMessage] = useState('');

  const submit = async (e) => {
    e.preventDefault();
    setErrorMessage('');

    if (city == '') {
      setErrorMessage('missing city value');
      return;
    }

    if (years <= 0) {
      setErrorMessage('invalid years value');
      return;
    }

    fetch(`${process.env.REACT_APP_WEATHER_API_URL}/windstatistics?city=${city}&years=${years}`, { 
      method: 'GET',
    })
      .then(async response => {
        const data = await response.json();

        if (response.ok) {
          return;
        }

        if (response.status > 500) {
          setErrorMessage('internal error. please, try again');
        } else {
          setErrorMessage(data.Message);
        }
        
        const error = (data && data.Message) || response.status;
        return Promise.reject(error);
      })
      .catch(error => {
        console.log('err', error.toString());
      });
  };

  return (
    <>            
      <main className={classes.main}>
        <div className={classes.input_container}> 
          <div className={classes.paper}>
            <form
              noValidate
              className={classes.form}
              onSubmit={submit}
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
                onChange={e=>setCity(e.target.value)}
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
                onChange={e=>setYears(e.target.value)}
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
              {errorMessage && <div className={classes.error}> 
                {errorMessage} 
              </div>}
            </form>
          </div> 
        </div> 
      </main> 
    </>
  );
}
