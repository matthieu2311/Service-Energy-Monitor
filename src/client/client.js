async function fetchCarbonIntensity() {
    try {                                                                   
        const response = await fetch('https://api.electricitymap.org/v3/carbon-intensity/history?zone=FR', {
            //Don't forget to change the zone in the url above if you do not want to monitor french electricity
            method: 'GET',
            headers: {
                'auth-token': '' //Put your own token here (creating one on the website is free)
            }
        });

        if (!response.ok) {
            throw new Error(`HTTP error! Status: ${response.status}`);
        }

        const data = await response.json();

        const filteredData = data.history.filter(entry => entry.isEstimated === false);

        const lastValidEntry = filteredData.pop(); 

        displayData(lastValidEntry);

    } catch (error) {
        console.error("Error fetching data:", error);
    }
}

function displayData(entry) {
    const containers = document.getElementsByClassName('button-div');
    
    if (!entry) {
        for (let container of containers){
            container.innerHTML = "<p>No valid data available.</p>";
        }
        return;
    }

    for(let container of containers){
        container.innerHTML = `<p>This server is using energy with a CO2 equivalent of <strong> ${entry.carbonIntensity} </strong> gCO<sub>2eq</sub>/kWh </p>
        <p><em>Data recolted at ${entry.datetime} with the api https://app.electricitymaps.com/</em>`
    }
    
}

// Call the function to fetch and display data
fetchCarbonIntensity();
