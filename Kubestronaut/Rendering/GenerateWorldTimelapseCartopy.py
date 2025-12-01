import cartopy.crs as ccrs
import geopandas as gpd
import matplotlib.pyplot as plt
from matplotlib.animation import FuncAnimation
import contextily as cx
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

# Set a valid projection if missing (WGS84 / EPSG:4326 for lat/lon)
if world.crs is None:
    world.set_crs(epsg=4326, inplace=True)

# Mapping between timeline countries and shapefile country names
country_name_mapping = {
    "United States": "United States of America",
    "UK": "United Kingdom",
    "Russia": "Russian Federation",
    # Add more mappings as necessary
}

# Track countries that have been "found" to color them light blue in subsequent frames
found_countries = set()

# Initialize the figure with Robinson projection
fig, ax = plt.subplots(figsize=(12, 8), subplot_kw={'projection': ccrs.Robinson()})

# Function to update the map
def update(frame):
    ax.clear()
    ax.set_extent([-180, 180, -90, 90])
    ax.set_axis_off()

    # Handle frames beyond the length of df
    if frame >= len(df):
        frame = len(df) - 1  # Prevent out-of-bounds error by limiting the frame to the last row

    # Get the current timestamp and country
    current_time = df.iloc[frame]['Timestamp']
    current_country = df.iloc[frame]['Country']
    mapped_country = country_name_mapping.get(current_country, current_country)

    # If the country is found, add it to the set of "discovered" countries
    if mapped_country in world['ADMIN'].values:
        found_countries.add(mapped_country)
    
    # Apply the color mapping for each country
    def color_country(country):
        if country == mapped_country:
            return '#08306b'  # Dark blue for the current country
        elif country in found_countries:
            return '#6baed6'  # Light blue for previous countries
        else:
            return '#f7fbff'  # Very light blue (background)

    # Apply the color mapping to the world DataFrame
    world['color'] = world['ADMIN'].apply(color_country)
    
    # Plot the countries with the specified color
    world.plot(ax=ax, color=world['color'], edgecolor='black', transform=ccrs.PlateCarree(), alpha=0.8)

    # Try to add a basemap for context; if unavailable, continue without it
    try:
        cx.add_basemap(ax, source=cx.providers.OpenStreetMap.Mapnik)
    except Exception:
        pass

    # Clear any previous text to avoid overlap and re-render fresh text
    fig.texts.clear()

    # Display the date and number of countries discovered below the map
    fig.text(0.5, 0.02, f"Date: {current_time.date()}", ha="center", fontsize=14)
    fig.text(0.5, 0.01, f"Number of Countries: {len(found_countries)}", ha="center", fontsize=12)

# Set the number of extra frames to hold the last image
extra_frames = 10

# Create the animation, with the last frame repeated
anim = FuncAnimation(fig, update, frames=len(df) + extra_frames, repeat=False)

# Save the animation with higher DPI for better quality
anim.save('better_timelapse_colored.gif', writer='pillow', dpi=300)

# To display the animation in a notebook (optional)
plt.show()

