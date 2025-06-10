# Serverless lottery scraper

A serverless web scraper written in Go that extracts lottery results from PDF files on the official Kerala Lottery website. The project leverages serverless architecture for scalability and is deployed on Vercel.

## Features

- **Scrapes Kerala Lottery PDFs:** Automatically fetches and parses lottery result PDFs from the official Kerala Lottery site.
- **Serverless Deployment:** Utilizes Vercel’s serverless platform for efficient, scalable execution.
- **Written in Go:** High-performance backend logic using Golang.
- **Classic Web Stack:** Supplementary UI and assets use JavaScript, HTML, and CSS.

## How It Works

1. The scraper fetches the latest PDF files from the Kerala Lottery results page.
2. Parses the PDFs using regex to extract winning numbers and relevant data.
3. Exposes a web endpoint (or API) for accessing the parsed data.

## Getting Started

### Deployment
If changes are pushed into the master branch they automatically get deployed on vercel.
Changes can be tested using the code in https://github.com/Joe-Albert-1703/lottery-scraper

## Configuration

- Update any environment variables as needed for your use case (e.g., target URLs, scraping intervals, etc.).
- See the project’s source code for additional configuration options.

## Usage

- Access the deployed endpoint to trigger a scrape or view results.
- Currently Deployed in: [serverless-lottery-scraper.vercel.app](https://serverless-lottery-scraper.vercel.app/)

## Contributing

Pull requests are welcome! For major changes, please open an issue first to discuss what you’d like to change.
