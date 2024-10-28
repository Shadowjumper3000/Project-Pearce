"""
Retrieve the next bus estimates for a given stop number.
"""

import requests
from bs4 import BeautifulSoup

# Test if the requests library is working
try:
    test_response = requests.get("https://www.google.com", timeout=10)
    print("Requests library is installed successfully!")
except requests.exceptions.RequestException as e:
    print("Error:", e)


def get_next_bus(stop_number):
    """
    Retrieve the next bus estimates for a given stop number.

    Args:
        stop_number (int): The stop number to retrieve estimates for.
    """
    url = f"https://www.emtmadrid.es/PMVVisor/pmv.aspx?stopnum={stop_number}&size=3"
    try:
        response = requests.get(url, timeout=10)
        response.raise_for_status()  # Raise an error for bad status codes
    except requests.exceptions.RequestException as e:
        print(f"Error retrieving data for stop {stop_number}: {e}")
        return

    soup = BeautifulSoup(response.text, 'html.parser')

    # Locate the table containing the bus estimates
    table = soup.find("table")

    if not table:
        print(f"No table found for stop {stop_number}.")
        return

    rows = table.find_all("tr")  # Get all rows in the table

    for row in rows[1:]:  # Skip header row
        cols = row.find_all("td")  # Get columns for each row
        if len(cols) >= 3:  # Ensure there are enough columns
            line = cols[0].text.strip()  # Line number
            direction = cols[1].text.strip()  # Direction
            time = cols[2].text.strip()  # Time
            print(f"Stop {stop_number} - Line: {line}, Direction: {direction}, Time: {time}")
        else:
            print(f"Row has insufficient columns for stop {stop_number}.")


if __name__ == "__main__":
    # Define stop numbers for home and gym
    home_stop_number = 1490
    gym_stop_number = 1778

    # Print estimates for both stops
    print("Bus estimates for home stop:")
    get_next_bus(home_stop_number)
    print("\nBus estimates for gym stop:")
    get_next_bus(gym_stop_number)
