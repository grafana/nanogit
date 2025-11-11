#!/usr/bin/env python3
"""Remove white background from logo and make it transparent."""

from PIL import Image

def make_transparent(input_path, output_path, threshold=240):
    """Convert white/light backgrounds to transparent.

    Args:
        input_path: Path to input image
        output_path: Path to save transparent image
        threshold: RGB value threshold for considering a pixel "white" (0-255)
    """
    # Open image and convert to RGBA
    img = Image.open(input_path).convert('RGBA')

    # Get pixel data
    pixels = img.load()
    width, height = img.size

    # Make white/light pixels transparent
    for y in range(height):
        for x in range(width):
            r, g, b, a = pixels[x, y]

            # If pixel is white or very light, make it transparent
            if r >= threshold and g >= threshold and b >= threshold:
                pixels[x, y] = (r, g, b, 0)  # Set alpha to 0 (transparent)

    # Save as PNG with transparency
    img.save(output_path, 'PNG')
    print(f"✓ Converted {input_path} to transparent PNG: {output_path}")

if __name__ == '__main__':
    # Convert the logo
    make_transparent(
        'docs/public/logo.png',
        'docs/public/logo.png',
        threshold=240
    )
    print("✓ Logo background removed successfully!")
