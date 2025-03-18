import matplotlib.pyplot as plt
import pandas as pd
from datetime import datetime

# List of timestamps as strings
timestamps = [
    "4/11/2024 20:39:55", "4/13/2024 7:16:38", "5/9/2024 2:47:02", 
    "5/15/2024 7:48:34", "5/28/2024 4:19:45", "6/10/2024 19:22:12", 
    "6/27/2024 15:03:46", "7/4/2024 19:55:08", "7/4/2024 20:32:45", 
    "7/22/2024 10:03:40", "7/23/2024 14:33:39", "8/1/2024 6:36:33", 
    "8/12/2024 19:49:03", "9/8/2024 21:55:00", "9/11/2024 3:23:49", 
    "9/24/2024 8:39:34", "9/24/2024 20:32:58", "9/27/2024 8:29:17", 
    "10/2/2024 9:04:21", "10/7/2024 21:52:18", "10/7/2024 22:29:25"
]

# Convert timestamps to datetime objects
timestamps_dt = pd.to_datetime(timestamps)

# Create a dataframe to store the data and add cumulative counts
df = pd.DataFrame({'timestamp': timestamps_dt})
df['kubestronauts'] = range(1, len(df) + 1)  # Cumulative count

# Add the origin date (March 22, 2024) with cumulative count of 0
origin = pd.to_datetime("2024-03-22")
df = pd.concat([pd.DataFrame({'timestamp': [origin], 'kubestronauts': [0]}), df]).reset_index(drop=True)

# Plot the cumulative distribution by date with months as x-axis
plt.figure(figsize=(10, 6))
plt.plot(df['timestamp'], df['kubestronauts'], marker='o', linestyle='-', color='b')

# Format the x-axis to show months
plt.gca().xaxis.set_major_formatter(plt.matplotlib.dates.DateFormatter('%b %Y'))

# Ensure all months are displayed, even those with no events
plt.gca().xaxis.set_major_locator(plt.matplotlib.dates.MonthLocator())

# Add titles and labels
plt.title('Cumulative Number of Kubestronauts Over Time (Origin: March 22, 2024)')
plt.xlabel('Date (Month Reference)')
plt.ylabel('Total Kubestronauts')

# Rotate x-axis labels for clarity
plt.xticks(rotation=45)

# Display grid
plt.grid(True)

# Save the plot as PNG and SVG
plt.savefig("kubestronauts_cumulative_with_origin.png", format="png", dpi=300)
plt.savefig("kubestronauts_cumulative_with_origin.svg", format="svg", dpi=300)

# Show the plot
plt.show()

