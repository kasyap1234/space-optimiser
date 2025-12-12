# 3D Bin Packing API

**Optimize your packing with advanced 3D bin packing algorithms**

This API solves the 3D bin packing problem - efficiently packing items of various sizes into containers (boxes) while minimizing wasted space. Perfect for logistics, warehouse management, shipping optimization, and inventory planning.

## 🚀 Key Features

- **Smart Packing Algorithm**: Uses advanced heuristics to maximize space utilization
- **Multiple Box Support**: Try different box sizes and automatically select the best fit
- **3D Visualization**: Get an interactive 3D view of your packing results
- **Rotation Support**: Items can be rotated in all 6 orientations for optimal fit
- **Real-time Results**: Fast computation with detailed statistics

## 📋 API Endpoints

### POST `/pack`

Pack items into boxes using an optimized 3D bin packing algorithm.

**Request Body:**

```json
{
  "items": [
    {
      "id": "item-1",
      "w": 10,
      "h": 10,
      "d": 10,
      "quantity": 2
    },
    {
      "id": "item-2",
      "w": 20,
      "h": 20,
      "d": 20,
      "quantity": 1
    }
  ],
  "boxes": [
    {
      "id": "small-box",
      "w": 15,
      "h": 15,
      "d": 15
    },
    {
      "id": "large-box",
      "w": 30,
      "h": 30,
      "d": 30
    }
  ]
}
```

**Request Parameters:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `items` | Array | Yes | List of items to pack |
| `items[].id` | String | Yes | Unique identifier for the item |
| `items[].w` | Integer | Yes | Width of the item |
| `items[].h` | Integer | Yes | Height of the item |
| `items[].d` | Integer | Yes | Depth of the item |
| `items[].quantity` | Integer | Yes | Number of this item to pack |
| `boxes` | Array | Yes | Available box types |
| `boxes[].id` | String | Yes | Unique identifier for the box |
| `boxes[].w` | Integer | Yes | Width of the box |
| `boxes[].h` | Integer | Yes | Height of the box |
| `boxes[].d` | Integer | Yes | Depth of the box |

**Response:**

```json
{
  "packed_boxes": [
    {
      "box_id": "large-box",
      "contents": [
        {
          "item_id": "item-2",
          "x": 0,
          "y": 0,
          "z": 0,
          "w": 20,
          "h": 20,
          "d": 20
        },
        {
          "item_id": "item-1",
          "x": 0,
          "y": 0,
          "z": 20,
          "w": 10,
          "h": 10,
          "d": 10
        },
        {
          "item_id": "item-1",
          "x": 0,
          "y": 10,
          "z": 20,
          "w": 10,
          "h": 10,
          "d": 10
        }
      ]
    }
  ],
  "unpacked_items": [],
  "total_volume": 27000,
  "utilization_percent": 25.93,
  "visualization_data_uri": "data:text/html;base64,PCFET0NUWVBFIGh0bWwgbGFuZz0iZW4iPjxoZWFkPi4uLjwvaGVhZD48Ym9keT4uLi48L2JvZHk+PC9odG1sPg==",
  "visualization_html": "<!DOCTYPE html>...[Full HTML with 3D visualization]..."
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `packed_boxes` | Array | List of boxes with packed items |
| `packed_boxes[].box_id` | String | ID of the box used |
| `packed_boxes[].contents` | Array | Items packed in this box |
| `packed_boxes[].contents[].item_id` | String | ID of the packed item |
| `packed_boxes[].contents[].x` | Integer | X coordinate of item position |
| `packed_boxes[].contents[].y` | Integer | Y coordinate of item position |
| `packed_boxes[].contents[].z` | Integer | Z coordinate of item position |
| `packed_boxes[].contents[].w` | Integer | Width of item (may be rotated) |
| `packed_boxes[].contents[].h` | Integer | Height of item (may be rotated) |
| `packed_boxes[].contents[].d` | Integer | Depth of item (may be rotated) |
| `unpacked_items` | Array | Items that couldn't fit in any box |
| `total_volume` | Integer | Total volume of all boxes used |
| `utilization_percent` | Float | Percentage of box space utilized |
| `visualization_data_uri` | String | Data URI for instant 3D visualization (paste into browser address bar) |
| `visualization_html` | String | Raw HTML string for saving as .html file and opening locally |

**Status Codes:**

- `200 OK`: Packing completed successfully
- `400 Bad Request`: Invalid request format or missing required fields
- `500 Internal Server Error`: Server error during processing

## 🎨 3D Visualization

Each packing result includes two visualization options:

### Option 1: Data URI (Instant Preview)
Copy the `visualization_data_uri` value from the response and paste it directly into your browser's address bar. The 3D visualization will load instantly without requiring any server.

### Option 2: HTML Download (Save Locally)
Copy the `visualization_html` value, save it as a `.html` file on your computer, and open it with your browser. This allows you to save the visualization for later reference.

**Visualization Features:**
- **Interactive Controls**: Rotate, pan, and zoom the 3D scene
- **Color-coded Items**: Each item has a unique color for easy identification
- **Detailed Stats**: View box count, item count, and utilization percentage
- **Professional UI**: Modern, responsive design with dark theme

**Controls:**
- **Left Click + Drag**: Rotate the view
- **Right Click + Drag**: Pan the camera
- **Scroll Wheel**: Zoom in/out

## 💡 Use Cases

### E-commerce & Shipping
Optimize package selection to reduce shipping costs and minimize material waste.

```json
{
  "items": [
    {"id": "product-A", "w": 15, "h": 10, "d": 8, "quantity": 3},
    {"id": "product-B", "w": 20, "h": 15, "d": 10, "quantity": 2}
  ],
  "boxes": [
    {"id": "small", "w": 30, "h": 20, "d": 20},
    {"id": "medium", "w": 40, "h": 30, "d": 30},
    {"id": "large", "w": 60, "h": 40, "d": 40}
  ]
}
```

### Warehouse Management
Plan storage layouts and optimize container loading.

```json
{
  "items": [
    {"id": "pallet-1", "w": 120, "h": 100, "d": 80, "quantity": 5},
    {"id": "pallet-2", "w": 100, "h": 120, "d": 80, "quantity": 3}
  ],
  "boxes": [
    {"id": "container-20ft", "w": 589, "h": 239, "d": 235},
    {"id": "container-40ft", "w": 1203, "h": 239, "d": 235}
  ]
}
```

### Moving & Relocation
Determine how many boxes or trucks you need for a move.

```json
{
  "items": [
    {"id": "box-books", "w": 40, "h": 30, "d": 30, "quantity": 10},
    {"id": "box-clothes", "w": 50, "h": 40, "d": 40, "quantity": 8},
    {"id": "furniture", "w": 200, "h": 100, "d": 80, "quantity": 2}
  ],
  "boxes": [
    {"id": "small-truck", "w": 300, "h": 200, "d": 150},
    {"id": "large-truck", "w": 600, "h": 250, "d": 200}
  ]
}
```

## 📊 Algorithm Details

The API uses an advanced 3D bin packing algorithm with the following features:

- **First-Fit Decreasing**: Items are sorted by volume (largest first) for better packing
- **Guillotine Heuristic**: Efficient space splitting and management
- **6-Rotation Support**: Items can be rotated in all orientations
- **Best-Fit Selection**: Chooses the box that minimizes wasted space
- **Space Merging**: Combines adjacent free spaces to reduce fragmentation

## ⚡ Performance

- **Fast Processing**: Typical response time < 100ms for standard requests
- **Scalable**: Handles hundreds of items and multiple box types
- **Optimized**: Uses efficient data structures and algorithms

## 🔒 Best Practices

1. **Provide Multiple Box Options**: The algorithm will select the most efficient box
2. **Use Realistic Dimensions**: Ensure all measurements are in the same unit (e.g., cm)
3. **Consider Weight Limits**: This API optimizes for volume only, not weight
4. **Check Unpacked Items**: Review the `unpacked_items` array for items that didn't fit
5. **Visualize Results**: Use the visualization data to verify packing accuracy

## 📝 Example Code

### cURL

```bash
curl -X POST https://space-optimiser.p.rapidapi.com/pack \
  -H "Content-Type: application/json" \
  -H "X-RapidAPI-Key: YOUR_API_KEY" \
  -H "X-RapidAPI-Host: space-optimiser.p.rapidapi.com" \
  -d '{
    "items": [
      {"id": "item-1", "w": 10, "h": 10, "d": 10, "quantity": 2}
    ],
    "boxes": [
      {"id": "box-1", "w": 30, "h": 30, "d": 30}
    ]
  }'
```

### Python

```python
import requests
import json

url = "https://space-optimiser.p.rapidapi.com/pack"
headers = {
    "Content-Type": "application/json",
    "X-RapidAPI-Key": "YOUR_API_KEY",
    "X-RapidAPI-Host": "space-optimiser.p.rapidapi.com"
}
payload = {
    "items": [
        {"id": "item-1", "w": 10, "h": 10, "d": 10, "quantity": 2}
    ],
    "boxes": [
        {"id": "box-1", "w": 30, "h": 30, "d": 30}
    ]
}

response = requests.post(url, json=payload, headers=headers)
result = response.json()

print(f"Utilization: {result['utilization_percent']}%")

# Option 1: Print the data URI for browser preview
print(f"Visualization Data URI: {result['visualization_data_uri'][:50]}...")

# Option 2: Save HTML to file
with open('visualization.html', 'w') as f:
    f.write(result['visualization_html'])
print("Visualization saved to visualization.html")
```

### JavaScript (Node.js)

```javascript
const axios = require('axios');
const fs = require('fs');

const options = {
  method: 'POST',
  url: 'https://space-optimiser.p.rapidapi.com/pack',
  headers: {
    'Content-Type': 'application/json',
    'X-RapidAPI-Key': 'YOUR_API_KEY',
    'X-RapidAPI-Host': 'space-optimiser.p.rapidapi.com'
  },
  data: {
    items: [
      {id: 'item-1', w: 10, h: 10, d: 10, quantity: 2}
    ],
    boxes: [
      {id: 'box-1', w: 30, h: 30, d: 30}
    ]
  }
};

axios.request(options)
  .then(response => {
    console.log(`Utilization: ${response.data.utilization_percent}%`);
    
    // Option 1: Log data URI
    console.log(`Visualization Data URI: ${response.data.visualization_data_uri.substring(0, 50)}...`);
    
    // Option 2: Save HTML to file
    fs.writeFileSync('visualization.html', response.data.visualization_html);
    console.log('Visualization saved to visualization.html');
  })
  .catch(error => console.error(error));
```

## ❓ FAQ

**Q: What units should I use for dimensions?**  
A: Use any consistent unit (cm, inches, mm, etc.). The algorithm works with relative sizes.

**Q: Can items be rotated?**  
A: Yes! The algorithm automatically tries all 6 possible rotations for each item.

**Q: What if not all items fit?**  
A: Items that don't fit are returned in the `unpacked_items` array. Consider providing larger boxes.

**Q: How do I view the visualization?**  
A: You have two options:
1. Copy the `visualization_data_uri` and paste it into your browser's address bar
2. Copy the `visualization_html`, save it as a `.html` file, and open it with your browser

**Q: Is there a limit on the number of items or boxes?**  
A: For optimal performance, we recommend keeping requests under 1000 items and 50 box types.

## 🆘 Support

Need help? Have questions? Contact us:
- **Email**: kasyap3103@gmail.com
- **RapidAPI Marketplace**: https://rapidapi.com/kasyap3103/api/space-optimiser
- **GitHub**: https://github.com/kasyap3103/space-optimiser

---

**Made with ❤️ for developers who need efficient packing solutions**
