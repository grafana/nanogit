#!/usr/bin/env python3
"""Remove white background from logo and make it transparent, while preserving interior."""

from PIL import Image, ImageDraw
import math

def make_transparent_with_boundary(input_path, output_path, threshold=250):
    """Convert outer white background to transparent, preserve center white.

    This removes only the very bright background pixels (threshold 250+)
    which are outside the main gradient circle, preserving the white
    interior of the cloud icon.

    Args:
        input_path: Path to input image
        output_path: Path to save transparent image
        threshold: RGB value threshold for "background white" (250-255)
    """
    # Open image and convert to RGBA
    img = Image.open(input_path).convert('RGBA')
    pixels = img.load()
    width, height = img.size

    # Calculate center and radius of the gradient circle
    center_x, center_y = width // 2, height // 2

    # Estimate the gradient circle radius (it doesn't go to edges)
    # The gradient appears to be about 70-80% of the image size
    radius = min(width, height) * 0.4

    # Make very bright pixels transparent, but only outside the main circle
    for y in range(height):
        for x in range(width):
            r, g, b, a = pixels[x, y]

            # Calculate distance from center
            distance = math.sqrt((x - center_x)**2 + (y - center_y)**2)

            # Only make pixels transparent if they're very bright (250+)
            # AND outside the gradient circle radius
            if r >= threshold and g >= threshold and b >= threshold and distance > radius:
                pixels[x, y] = (r, g, b, 0)

    # Save as PNG with transparency
    img.save(output_path, 'PNG')
    print(f"✓ Converted {input_path} to transparent PNG: {output_path}")
    print(f"  - Removed outer background (threshold: {threshold})")
    print(f"  - Preserved interior within radius: {radius:.1f}px")

if __name__ == '__main__':
    # Convert the logo, preserving interior white
    make_transparent_with_boundary(
        'docs/public/logo.png',
        'docs/public/logo.png',
        threshold=250
    )
    print("✓ Logo background removed successfully!")
