import geopandas as gpd
import matplotlib.pyplot as plt
from matplotlib.animation import FuncAnimation
import pandas as pd
import json

# Load the data from the JSON file
with open('countries_timeline.json', 'r') as f:
    data = json.load(f)

# Convert to a DataFrame and sort by timestamp
df = pd.DataFrame(data)
df['Timestamp'] = pd.to_datetime(df['Timestamp'])
df = df.sort_values(by='Timestamp')

# Load the world shapefile from the local directory
shapefile_path = 'countries_shape/ne_10m_admin_0_countries.shp'
world = gpd.read_file(shapefile_path)

# Mapping between timeline countries and shapefile country names
country_name_mapping = {
    "United States": "United States of America",
    "UK": "United Kingdom",
    "Russia": "Russian Federation",
    # Add more mappings as necessary
}

# Track countries that have been "found" to color them light blue in subsequent frames
found_countries = set()

# Initialize the figure and axis for the animation
fig, ax = plt.subplots(figsize=(12, 8))

# Function to update the map at each frame
def update(frame):
    ax.clear()
    
    # Remove the axes for a cleaner look
    ax.set_axis_off()
    
    # Get the current timestamp and country
    current_time = df.iloc[frame]['Timestamp']
    current_country = df.iloc[frame]['Country']

    # Map the country if necessary
    mapped_country = country_name_mapping.get(current_country, current_country)
    
    # Check if the country exists in the shapefile
    if mapped_country in world['ADMIN'].values:
        print(f"Timestamp: {current_time.date()}, Country: {current_country} (mapped as '{mapped_country}'), Status: Found")
        found_countries.add(mapped_country)  # Add the country to the found set
    else:
        print(f"WARNING: Timestamp: {current_time.date()}, Country: {current_country} (mapped as '{mapped_country}'), Status: Not Found")
    
    # Apply the color mapping
    def color_country(country):
        if country == mapped_country:
            return 'darkblue'  # Dark blue for the current country being processed
        elif country in found_countries:
            return 'lightblue'  # Light blue for previously found countries
        else:
            return 'white'  # White for countries not yet processed

    # Color the map based on the country status
    world['color'] = world['ADMIN'].apply(color_country)
    
    # Plot the updated map
    world.plot(ax=ax, color=world['color'], edgecolor='black')
    
    # Add title with the current timestamp
    plt.title(f'Countries up to {current_time.date()}', fontsize=16)

# Create the animation
anim = FuncAnimation(fig, update, frames=len(df), repeat=False)

# Save the animation as a gif using Pillow
anim.save('timelapse_countries.gif', writer='pillow')

# To display the animation in a notebook (optional)
plt.show()

