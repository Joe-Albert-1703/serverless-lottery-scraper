document.addEventListener('DOMContentLoaded', function() {
    fetchResults();
    fetchLotteries();

    const ticketForm = document.getElementById('ticket-form');
    ticketForm.addEventListener('submit', function(event) {
        event.preventDefault();
        checkTickets();
    });
});

function fetchResults() {
    fetch('/api/v1/all_results')
        .then(response => response.json())
        .then(data => {
            const resultsContainer = document.getElementById('results-container');
            resultsContainer.innerHTML = '';

            fetch('/api/v1/list_lotteries')
                .then(response => response.json())
                .then(lotteries => {
                    lotteries.forEach(lottery => {
                        const lotteryName = lottery.lottery_name;
                        const lotteryResults = data.results[lotteryName];

                        if (lotteryResults) {
                            const lotteryDiv = createLotteryDiv(lotteryName, lotteryResults);
                            resultsContainer.appendChild(lotteryDiv);
                        }
                    });
                })
                .catch(error => console.error('Error fetching lotteries:', error));
        })
        .catch(error => console.error('Error fetching results:', error));
}

function createLotteryDiv(lotteryName, results) {
    const lotteryDiv = document.createElement('div');
    lotteryDiv.classList.add('lottery-box');
    lotteryDiv.style.backgroundColor = '#3E3E3E';
    
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

function fetchLotteries() {
    fetch('/api/v1/list_lotteries')
        .then(response => response.json())
        .then(data => {
            const lotteriesContainer = document.getElementById('lotteries-container');
            lotteriesContainer.innerHTML = '';

            data.forEach(lottery => {
                const lotteryDiv = document.createElement('div');
                lotteryDiv.classList.add('lottery');
                lotteryDiv.innerHTML = `<strong>${lottery.lottery_name}</strong> (${lottery.lottery_date}) <a href="${lottery.pdf_link}" target="_blank">View PDF</a>`;
                lotteriesContainer.appendChild(lotteryDiv);
            });
        })
        .catch(error => console.error('Error fetching lotteries:', error));
}

function checkTickets() {
    const ticketsInput = document.getElementById('tickets').value;
    const tickets = ticketsInput.split(',').map(ticket => ticket.trim());

    fetch('/api/v1/check_tickets', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify(tickets),
    })
        .then(response => response.json())
        .then(data => {
            const winnersContainer = document.getElementById('winners-container');
            winnersContainer.innerHTML = '';

            if (Object.keys(data).length > 0) {
                Object.entries(data).forEach(([position, lotteries]) => {
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
        })
        .catch(error => console.error('Error checking tickets:', error));
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
