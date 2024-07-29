let lotteryResultsData = {}; // Global variable to store lottery results

document.addEventListener('DOMContentLoaded', function() {
    fetchAllData();

    const ticketForm = document.getElementById('ticket-form');
    ticketForm.addEventListener('submit', function(event) {
        event.preventDefault();
        checkTickets();
    });
});

function fetchAllData() {
    // Fetch both results and lotteries in parallel
    Promise.all([
        fetch('/api/v1/all_results').then(response => response.json()),
        fetch('/api/v1/list_lotteries').then(response => response.json())
    ]).then(([resultsData, lotteriesData]) => {
        // Process results
        lotteryResultsData = resultsData.results; // Store the results data for later use
        const resultsContainer = document.getElementById('results-container');
        resultsContainer.innerHTML = '';

        lotteriesData.forEach(lottery => {
            const lotteryName = lottery.lottery_name;
            const lotteryResults = lotteryResultsData[lotteryName];

            if (lotteryResults) {
                const lotteryDiv = createLotteryDiv(lotteryName, lotteryResults);
                resultsContainer.appendChild(lotteryDiv);
            }
        });

        // Process lotteries list
        const lotteriesContainer = document.getElementById('lotteries-container');
        lotteriesContainer.innerHTML = '';

        lotteriesData.forEach(lottery => {
            const lotteryDiv = document.createElement('div');
            lotteryDiv.classList.add('lottery');
            lotteryDiv.innerHTML = `<strong>${lottery.lottery_name}</strong> (${lottery.lottery_date}) <a href="${lottery.pdf_link}" target="_blank">View PDF</a>`;
            lotteriesContainer.appendChild(lotteryDiv);
        });

    }).catch(error => console.error('Error fetching data:', error));
}

function checkTickets() {
    const ticketsInput = document.getElementById('tickets').value;
    const tickets = ticketsInput.split(',').map(ticket => ticket.trim());
    const winnersContainer = document.getElementById('winners-container');
    winnersContainer.innerHTML = '';

    const winners = {};

    tickets.forEach(ticket => {
        Object.entries(lotteryResultsData).forEach(([lottery, results]) => {
            Object.entries(results).forEach(([position, numbers]) => {
                if (numbers.includes(ticket)) {
                    if (!winners[position]) {
                        winners[position] = {};
                    }
                    if (!winners[position][lottery]) {
                        winners[position][lottery] = [];
                    }
                    winners[position][lottery].push(ticket);
                }
            });
        });
    });

    if (Object.keys(winners).length > 0) {
        Object.entries(winners).forEach(([position, lotteries]) => {
            const positionDiv = document.createElement('div');
            positionDiv.classList.add('winner');

            const title = document.createElement('h3');
            title.textContent = `Position: ${position}`;
            positionDiv.appendChild(title);

            Object.entries(lotteries).forEach(([lottery, winningTickets]) => {
                const lotteryDiv = document.createElement('div');
                lotteryDiv.innerHTML = `<strong>${lottery}:</strong> ${winningTickets.join(', ')}`;
                positionDiv.appendChild(lotteryDiv);
            });

            winnersContainer.appendChild(positionDiv);
        });
    } else {
        winnersContainer.textContent = 'No winning tickets';
    }
}

function createLotteryDiv(lotteryName, results) {
    const lotteryDiv = document.createElement('div');
    lotteryDiv.classList.add('lottery-box');
    lotteryDiv.style.backgroundColor = '#0E0E0E';
    
    const title = document.createElement('div');
    title.classList.add('lottery-title');
    title.textContent = `Lottery: ${lotteryName}`;
    title.classList.add('collapsed');
    title.addEventListener('click', () => toggleCollapse(title));
    lotteryDiv.appendChild(title);

    Object.entries(results).forEach(([position, numbers]) => {
        if (position === "Series") {
            return;
        }

        const positionDiv = document.createElement('div');
        positionDiv.classList.add('position-box', 'collapsed');
        positionDiv.innerHTML = `<strong>${position}:</strong>`;

        const numbersDiv = document.createElement('div');
        numbersDiv.classList.add('number-box');
        numbers.forEach(number => {
            const numberBox = document.createElement('div');
            numberBox.textContent = number;
            numbersDiv.appendChild(numberBox);
        });

        positionDiv.appendChild(numbersDiv);
        lotteryDiv.appendChild(positionDiv);
    });

    return lotteryDiv;
}

function toggleCollapse(element) {
    const isCollapsed = element.classList.contains('collapsed');

    document.querySelectorAll('.lottery-title, .position-box').forEach(el => el.classList.add('collapsed'));

    if (isCollapsed) {
        element.classList.remove('collapsed');
        const content = element.nextElementSibling;
        if (content) {
            content.classList.remove('collapsed');
        }
    }
}
