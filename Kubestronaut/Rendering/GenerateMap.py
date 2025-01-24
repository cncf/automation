import geopandas as gpd
import matplotlib.pyplot as plt
from adjustText import adjust_text

# Data for cities with latitudes, longitudes, and weights
cities = [
    {"city": "Aichi", "weight": 1, "lat": 34.9718, "lon": 137.0844, "x_offset": 1.0, "y_offset": 0.5},
    {"city": "Fukuoka", "weight": 1, "lat": 33.6064, "lon": 130.4181, "x_offset": -5.0, "y_offset": 0.0},
    {"city": "Ichikawa", "weight": 1, "lat": 35.7216, "lon": 139.9245, "x_offset": -1.5, "y_offset": 0.8},
    {"city": "Kitakyushu-shi", "weight": 1, "lat": 33.8836, "lon": 130.8758, "x_offset": -2.0, "y_offset": 1.0},
    {"city": "Mie", "weight": 1, "lat": 34.7303, "lon": 136.5086, "x_offset": 0.0, "y_offset": -0.6},
    {"city": "Osaka", "weight": 1, "lat": 34.6937, "lon": 135.5023, "x_offset": -0.5, "y_offset": 0.6},
    {"city": "Tokyo", "weight": 9, "lat": 35.6762, "lon": 139.6503, "x_offset": -1.5, "y_offset": 0.25},
    {"city": "Yokohama", "weight": 2, "lat": 35.4437, "lon": 139.6380, "x_offset": 1.5, "y_offset": -0.8}
]

# Data for grouped close cities (Tokyo, Yokohama, Kanagawa, Aichi)
grouped_cities = {"city": "", "weight": 17, "lat": 35.5, "lon": 139.6}

# Path to the 10m resolution shapefile
shapefile_path = './countries_shape/ne_10m_admin_0_countries.shp'

# Load the shapefile
world = gpd.read_file(shapefile_path)

# Filter for Japan only
japan = world[world['ADMIN'] == 'Japan']

# Create the plot
fig, ax = plt.subplots(figsize=(12, 12), dpi=300)

# Fill the sea with light blue color
ax.set_facecolor('#e6f7ff')  # Light blue for the sea

# Plot Japan with a very light color
japan.plot(ax=ax, color='#d9f2e6', edgecolor='black')  # Very light green for Japan with black boundaries

# Plot each city as a circle (excluding grouped cities)
texts = []  # Store text objects for adjustment
for city in cities:
    if city["city"] not in ["Tokyo", "Yokohama", "Aichi", "Ichikawa"]:  # Exclude Tokyo, Yokohama, Aichi and Ichikawa from individual plotting
        # Plot city circle
        ax.scatter(city["lon"], city["lat"], s=city["weight"] * 70, c='blue', alpha=0.6)
        # Add city label with offsets into the sea
        text = ax.text(city["lon"] + city["x_offset"], city["lat"] + city["y_offset"], 
                       f'{city["city"]} ({city["weight"]})', fontsize=12, ha='left', va='center')
        texts.append(text)
# Plot the grouped city circle for Tokyo, Yokohama, Kanagawa, and Aichi, "Ichikawa"
ax.scatter(grouped_cities["lon"], grouped_cities["lat"], s=grouped_cities["weight"] * 100, 
           c='darkblue', alpha=0.6, label='Greater Tokyo Area')

# Add a decomposition legend below the vertical position of Mie
ax.text(140.5, 31.0,  # Adjusted vertical position to be below Mie
        'Greater Tokyo Area Decomposition:\n'
        '- Tokyo: 9\n'
        '- Yokohama: 2\n'
        '- Kanagawa: 4\n'
        '- Ichikawa: 1\n'
        '- Aichi: 1', fontsize=12, bbox=dict(facecolor='white', alpha=0.7))

# Adjust text positions to avoid overlap using arrows
#adjust_text(texts, arrowprops=dict(arrowstyle='->', color='black', lw=0.8), 
#            precision=0.1, only_move={'text': 'xy'})

# Set titles and labels
ax.set_title('Kubestronauts in Japan', fontsize=20)
ax.set_xlabel('Longitude', fontsize=12)
ax.set_ylabel('Latitude', fontsize=12)

# Save as high-quality PNG and SVG
plt.savefig("kubestronauts_in_japan_grouped.png", format="png", dpi=300)
plt.savefig("kubestronauts_in_japan_grouped.svg", format="svg", dpi=300)

