import { Button, TextField } from '@material-ui/core';
import useStyles from './styles';
import { useState } from 'react';
import { LineChart, Line, CartesianGrid, XAxis, YAxis } from 'recharts';

export default function WeatherStatistics() {
  const classes = useStyles();

  const [city, setCity] = useState('');
  const [years, setYears] = useState(0);

  const [dataPoints, setDataPoints] = useState([]);

  const [errorMessage, setErrorMessage] = useState('');

  const submit = async (e) => {
    e.preventDefault();
    setErrorMessage('');
    setDataPoints([]);

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
          let windStats = [];

          for (let i = 0; i < data.length; i++) {
            windStats.push({
              year: data[i].Year,
              speed: data[i].Speed
            });
          }

          setDataPoints(windStats);
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
        <div className={classes.container} > 
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
                  label="Last years amount"
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
          {dataPoints.length > 0 && <div className={classes.chart}> 
            <LineChart width={900} height={300} data={dataPoints}>
              <Line type="monotone" dataKey="speed" stroke="violet" />
              <CartesianGrid stroke="white" />
              <XAxis dataKey="year" />
              <YAxis />
            </LineChart>
          </div>}
        </div>
      </main> 
    </>
  );
}
