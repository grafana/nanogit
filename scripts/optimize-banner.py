#!/usr/bin/env python3
"""Optimize banner images for web - reduce size and improve loading speed."""

from PIL import Image
import os

def optimize_banner(input_path, output_path, max_width=1200, quality=85):
    """Resize and optimize banner image for web.

    Args:
        input_path: Path to input banner image
        output_path: Path to save optimized image
        max_width: Maximum width in pixels (maintains aspect ratio)
        quality: PNG optimization level (1-100, higher = better quality)
    """
    # Open image
    img = Image.open(input_path)
    original_size = os.path.getsize(input_path) / 1024  # KB

    print(f"Original: {img.size[0]}x{img.size[1]} ({original_size:.1f} KB)")

    # Calculate new dimensions maintaining aspect ratio
    if img.size[0] > max_width:
        ratio = max_width / img.size[0]
        new_width = max_width
        new_height = int(img.size[1] * ratio)

        # Resize with high-quality resampling
        img = img.resize((new_width, new_height), Image.Resampling.LANCZOS)
        print(f"Resized to: {new_width}x{new_height}")
    else:
        print("Image already within size limits")

    # Optimize and save as PNG
    # Using optimize=True and reducing colors can significantly reduce file size
    img.save(output_path, 'PNG', optimize=True)

    new_size = os.path.getsize(output_path) / 1024  # KB
    reduction = ((original_size - new_size) / original_size) * 100

    print(f"Optimized: {new_size:.1f} KB (reduced by {reduction:.1f}%)")
    print(f"✓ Saved to {output_path}")

if __name__ == '__main__':
    # Optimize both banner locations
    print("Optimizing .github/assets/banner.png...")
    optimize_banner(
        '.github/assets/banner.png',
        '.github/assets/banner.png',
        max_width=1200
    )

    print("\nOptimizing docs/public/banner.png...")
    optimize_banner(
        'docs/public/banner.png',
        'docs/public/banner.png',
        max_width=1200
    )

    print("\n✓ Banner optimization complete!")
