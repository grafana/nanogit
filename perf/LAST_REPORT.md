# 🚀 Performance Benchmark Report

**Generated:** 2025-06-19T10:12:56+02:00  
**Total Benchmarks:** 168

## 📊 Performance Overview

| Operation | Speed Winner | Duration | In-Memory Winner | Memory Usage |
|-----------|--------------|----------|------------------|-------------|
| BulkCreateFiles_bulk_1000_files_medium | 🚀 nanogit | 798.7ms | 🐹 go-git | 29.3 MB |
| BulkCreateFiles_bulk_1000_files_small | 🚀 nanogit | 805.5ms | 🐹 go-git | 5.5 MB |
| BulkCreateFiles_bulk_100_files_medium | 🚀 nanogit | 105.2ms | 💚 nanogit | 5.3 MB |
| BulkCreateFiles_bulk_100_files_small | 🚀 nanogit | 97.1ms | 💚 nanogit | 3.2 MB |
| CompareCommits_adjacent_commits_large | 🚀 nanogit | 124.4ms | 💚 nanogit | 4.2 MB |
| CompareCommits_adjacent_commits_medium | 🚀 nanogit | 102.1ms | 💚 nanogit | 1.5 MB |
| CompareCommits_adjacent_commits_small | 🐹 go-git | 69.5ms | 💚 nanogit | 544.3 KB |
| CompareCommits_adjacent_commits_xlarge | 🚀 nanogit | 185.1ms | 💚 nanogit | 15.5 MB |
| CompareCommits_few_commits_large | 🚀 nanogit | 221.7ms | 💚 nanogit | 3.7 MB |
| CompareCommits_few_commits_medium | 🚀 nanogit | 183.6ms | 💚 nanogit | 1.8 MB |
| CompareCommits_few_commits_small | 🐹 go-git | 71.4ms | 💚 nanogit | 296.9 KB |
| CompareCommits_few_commits_xlarge | 🚀 nanogit | 324.4ms | 💚 nanogit | 14.5 MB |
| CompareCommits_max_commits_large | 🚀 nanogit | 324.7ms | 💚 nanogit | 3.1 MB |
| CompareCommits_max_commits_medium | 🚀 nanogit | 252.7ms | 💚 nanogit | 1.1 MB |
| CompareCommits_max_commits_small | 🐹 go-git | 74.1ms | 💚 nanogit | 581.4 KB |
| CompareCommits_max_commits_xlarge | 🚀 nanogit | 520.5ms | 💚 nanogit | 15.0 MB |
| CreateFile_large_repo | 🚀 nanogit | 58.8ms | 💚 nanogit | 4.2 MB |
| CreateFile_medium_repo | 🚀 nanogit | 51.0ms | 💚 nanogit | 2.3 MB |
| CreateFile_small_repo | 🚀 nanogit | 51.1ms | 💚 nanogit | 1.6 MB |
| CreateFile_xlarge_repo | 🚀 nanogit | 86.4ms | 💚 nanogit | 10.8 MB |
| DeleteFile_large_repo | 🚀 nanogit | 58.9ms | 💚 nanogit | 3.9 MB |
| DeleteFile_medium_repo | 🚀 nanogit | 50.5ms | 💚 nanogit | 2.1 MB |
| DeleteFile_small_repo | 🚀 nanogit | 46.7ms | 💚 nanogit | 1.2 MB |
| DeleteFile_xlarge_repo | 🚀 nanogit | 83.9ms | 💚 nanogit | 10.0 MB |
| GetFlatTree_large_tree | 🚀 nanogit | 39.8ms | 💚 nanogit | 2.0 MB |
| GetFlatTree_medium_tree | 🚀 nanogit | 34.5ms | 💚 nanogit | 1.0 MB |
| GetFlatTree_small_tree | 🚀 nanogit | 44.6ms | 💚 nanogit | 35.5 KB |
| GetFlatTree_xlarge_tree | 🚀 nanogit | 67.6ms | 💚 nanogit | 12.7 MB |
| UpdateFile_large_repo | 🚀 nanogit | 59.1ms | 💚 nanogit | 4.2 MB |
| UpdateFile_medium_repo | 🚀 nanogit | 49.2ms | 💚 nanogit | 2.9 MB |
| UpdateFile_small_repo | 🚀 nanogit | 48.7ms | 💚 nanogit | 1.9 MB |
| UpdateFile_xlarge_repo | 🚀 nanogit | 81.8ms | 💚 nanogit | 11.9 MB |

## ⚡ Duration Comparison

| Operation | git-cli | go-git | nanogit |
|-----------|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | 10.39s 🐌 | 70.99s 🐌 | 798.7ms 🏆 |
| BulkCreateFiles_bulk_1000_files_small | 9.95s 🐌 | 18.18s 🐌 | 805.5ms 🏆 |
| BulkCreateFiles_bulk_100_files_medium | 1.66s 🐌 | 6.56s 🐌 | 105.2ms 🏆 |
| BulkCreateFiles_bulk_100_files_small | 1.29s 🐌 | 901.0ms 🐌 | 97.1ms 🏆 |
| CompareCommits_adjacent_commits_large | 1.38s 🐌 | 2.63s 🐌 | 124.4ms 🏆 |
| CompareCommits_adjacent_commits_medium | 562.2ms 🐌 | 416.4ms | 102.1ms 🏆 |
| CompareCommits_adjacent_commits_small | 348.7ms 🐌 | 69.5ms 🏆 | 82.2ms ✅ |
| CompareCommits_adjacent_commits_xlarge | 5.60s 🐌 | 20.45s 🐌 | 185.1ms 🏆 |
| CompareCommits_few_commits_large | 1.50s 🐌 | 2.62s 🐌 | 221.7ms 🏆 |
| CompareCommits_few_commits_medium | 516.4ms | 418.9ms | 183.6ms 🏆 |
| CompareCommits_few_commits_small | 351.1ms | 71.4ms 🏆 | 158.6ms |
| CompareCommits_few_commits_xlarge | 5.71s 🐌 | 20.51s 🐌 | 324.4ms 🏆 |
| CompareCommits_max_commits_large | 1.37s | 2.61s 🐌 | 324.7ms 🏆 |
| CompareCommits_max_commits_medium | 480.7ms ✅ | 413.5ms ✅ | 252.7ms 🏆 |
| CompareCommits_max_commits_small | 348.0ms | 74.1ms 🏆 | 226.5ms |
| CompareCommits_max_commits_xlarge | 7.88s 🐌 | 20.91s 🐌 | 520.5ms 🏆 |
| CreateFile_large_repo | 1.76s 🐌 | 2.94s 🐌 | 58.8ms 🏆 |
| CreateFile_medium_repo | 967.7ms 🐌 | 513.4ms 🐌 | 51.0ms 🏆 |
| CreateFile_small_repo | 819.2ms 🐌 | 111.9ms | 51.1ms 🏆 |
| CreateFile_xlarge_repo | 8.82s 🐌 | 22.65s 🐌 | 86.4ms 🏆 |
| DeleteFile_large_repo | 1.75s 🐌 | 2.95s 🐌 | 58.9ms 🏆 |
| DeleteFile_medium_repo | 965.4ms 🐌 | 505.5ms 🐌 | 50.5ms 🏆 |
| DeleteFile_small_repo | 790.2ms 🐌 | 105.0ms | 46.7ms 🏆 |
| DeleteFile_xlarge_repo | 6.86s 🐌 | 22.56s 🐌 | 83.9ms 🏆 |
| GetFlatTree_large_tree | 1.14s 🐌 | 2.79s 🐌 | 39.8ms 🏆 |
| GetFlatTree_medium_tree | 506.0ms 🐌 | 455.5ms 🐌 | 34.5ms 🏆 |
| GetFlatTree_small_tree | 395.7ms 🐌 | 73.2ms ✅ | 44.6ms 🏆 |
| GetFlatTree_xlarge_tree | 5.21s 🐌 | 20.47s 🐌 | 67.6ms 🏆 |
| UpdateFile_large_repo | 1.77s 🐌 | 2.96s 🐌 | 59.1ms 🏆 |
| UpdateFile_medium_repo | 950.8ms 🐌 | 503.1ms 🐌 | 49.2ms 🏆 |
| UpdateFile_small_repo | 789.0ms 🐌 | 107.6ms | 48.7ms 🏆 |
| UpdateFile_xlarge_repo | 8.59s 🐌 | 22.52s 🐌 | 81.8ms 🏆 |

## 💾 Memory Usage Comparison

*Note: git-cli uses disk storage rather than keeping data in memory, so memory comparisons focus on in-memory clients (nanogit vs go-git)*

| Operation | git-cli | go-git | nanogit |
|-----------|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | 1.0 MB 💾 | 29.3 MB 🏆 | 179.3 MB 🔥 |
| BulkCreateFiles_bulk_1000_files_small | 121.3 KB 💾 | 5.5 MB 🏆 | 128.8 MB 🔥 |
| BulkCreateFiles_bulk_100_files_medium | -226264 B 💾 | 33.4 MB 🔥 | 5.3 MB 🏆 |
| BulkCreateFiles_bulk_100_files_small | 1.3 MB 💾 | 4.8 MB ✅ | 3.2 MB 🏆 |
| CompareCommits_adjacent_commits_large | 72.8 KB 💾 | 186.7 MB 🔥 | 4.2 MB 🏆 |
| CompareCommits_adjacent_commits_medium | 72.4 KB 💾 | 38.6 MB 🔥 | 1.5 MB 🏆 |
| CompareCommits_adjacent_commits_small | 72.8 KB 💾 | 3.7 MB 🔥 | 544.3 KB 🏆 |
| CompareCommits_adjacent_commits_xlarge | 73.7 KB 💾 | 1.7 GB 🔥 | 15.5 MB 🏆 |
| CompareCommits_few_commits_large | 72.8 KB 💾 | 209.3 MB 🔥 | 3.7 MB 🏆 |
| CompareCommits_few_commits_medium | 72.8 KB 💾 | 39.3 MB 🔥 | 1.8 MB 🏆 |
| CompareCommits_few_commits_small | 72.8 KB 💾 | 3.1 MB 🔥 | 296.9 KB 🏆 |
| CompareCommits_few_commits_xlarge | 72.8 KB 💾 | 1.5 GB 🔥 | 14.5 MB 🏆 |
| CompareCommits_max_commits_large | 73.6 KB 💾 | 209.2 MB 🔥 | 3.1 MB 🏆 |
| CompareCommits_max_commits_medium | 73.2 KB 💾 | 40.1 MB 🔥 | 1.1 MB 🏆 |
| CompareCommits_max_commits_small | 72.4 KB 💾 | 3.1 MB 🔥 | 581.4 KB 🏆 |
| CompareCommits_max_commits_xlarge | 72.6 KB 💾 | 1.5 GB 🔥 | 15.0 MB 🏆 |
| CreateFile_large_repo | 139.4 KB 💾 | 214.9 MB 🔥 | 4.2 MB 🏆 |
| CreateFile_medium_repo | 139.6 KB 💾 | 44.6 MB 🔥 | 2.3 MB 🏆 |
| CreateFile_small_repo | 139.3 KB 💾 | 5.0 MB | 1.6 MB 🏆 |
| CreateFile_xlarge_repo | 139.7 KB 💾 | 2.1 GB 🔥 | 10.8 MB 🏆 |
| DeleteFile_large_repo | 139.4 KB 💾 | 253.4 MB 🔥 | 3.9 MB 🏆 |
| DeleteFile_medium_repo | 139.3 KB 💾 | 43.3 MB 🔥 | 2.1 MB 🏆 |
| DeleteFile_small_repo | 139.9 KB 💾 | 3.4 MB | 1.2 MB 🏆 |
| DeleteFile_xlarge_repo | 139.4 KB 💾 | 2.1 GB 🔥 | 10.0 MB 🏆 |
| GetFlatTree_large_tree | 3.2 MB 💾 | 206.7 MB 🔥 | 2.0 MB 🏆 |
| GetFlatTree_medium_tree | 743.1 KB 💾 | 47.9 MB 🔥 | 1.0 MB 🏆 |
| GetFlatTree_small_tree | 157.6 KB 💾 | 3.2 MB 🔥 | 35.5 KB 🏆 |
| GetFlatTree_xlarge_tree | 18.7 MB 💾 | 1.7 GB 🔥 | 12.7 MB 🏆 |
| UpdateFile_large_repo | 138.8 KB 💾 | 216.1 MB 🔥 | 4.2 MB 🏆 |
| UpdateFile_medium_repo | 138.6 KB 💾 | 50.1 MB 🔥 | 2.9 MB 🏆 |
| UpdateFile_small_repo | 138.8 KB 💾 | 3.4 MB ✅ | 1.9 MB 🏆 |
| UpdateFile_xlarge_repo | 138.5 KB 💾 | 2.1 GB 🔥 | 11.9 MB 🏆 |

## 🎯 Nanogit Performance Analysis

### ⚡ Speed Comparison

| Operation | vs git-cli | vs go-git |
|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | 13.0x faster 🚀 | 88.9x faster 🚀 |
| BulkCreateFiles_bulk_1000_files_small | 12.4x faster 🚀 | 22.6x faster 🚀 |
| BulkCreateFiles_bulk_100_files_medium | 15.8x faster 🚀 | 62.4x faster 🚀 |
| BulkCreateFiles_bulk_100_files_small | 13.3x faster 🚀 | 9.3x faster 🚀 |
| CompareCommits_adjacent_commits_large | 11.1x faster 🚀 | 21.1x faster 🚀 |
| CompareCommits_adjacent_commits_medium | 5.5x faster 🚀 | 4.1x faster 🚀 |
| CompareCommits_adjacent_commits_small | 4.2x faster 🚀 | 1.2x slower 🐌 |
| CompareCommits_adjacent_commits_xlarge | 30.2x faster 🚀 | 110.5x faster 🚀 |
| CompareCommits_few_commits_large | 6.8x faster 🚀 | 11.8x faster 🚀 |
| CompareCommits_few_commits_medium | 2.8x faster 🚀 | 2.3x faster 🚀 |
| CompareCommits_few_commits_small | 2.2x faster 🚀 | 2.2x slower 🐌 |
| CompareCommits_few_commits_xlarge | 17.6x faster 🚀 | 63.2x faster 🚀 |
| CompareCommits_max_commits_large | 4.2x faster 🚀 | 8.0x faster 🚀 |
| CompareCommits_max_commits_medium | 1.9x faster ✅ | 1.6x faster ✅ |
| CompareCommits_max_commits_small | 1.5x faster ✅ | 3.1x slower 🐌 |
| CompareCommits_max_commits_xlarge | 15.1x faster 🚀 | 40.2x faster 🚀 |
| CreateFile_large_repo | 30.0x faster 🚀 | 50.0x faster 🚀 |
| CreateFile_medium_repo | 19.0x faster 🚀 | 10.1x faster 🚀 |
| CreateFile_small_repo | 16.0x faster 🚀 | 2.2x faster 🚀 |
| CreateFile_xlarge_repo | 102.1x faster 🚀 | 262.1x faster 🚀 |
| DeleteFile_large_repo | 29.7x faster 🚀 | 50.2x faster 🚀 |
| DeleteFile_medium_repo | 19.1x faster 🚀 | 10.0x faster 🚀 |
| DeleteFile_small_repo | 16.9x faster 🚀 | 2.2x faster 🚀 |
| DeleteFile_xlarge_repo | 81.7x faster 🚀 | 268.7x faster 🚀 |
| GetFlatTree_large_tree | 28.6x faster 🚀 | 70.0x faster 🚀 |
| GetFlatTree_medium_tree | 14.7x faster 🚀 | 13.2x faster 🚀 |
| GetFlatTree_small_tree | 8.9x faster 🚀 | 1.6x faster ✅ |
| GetFlatTree_xlarge_tree | 77.1x faster 🚀 | 302.7x faster 🚀 |
| UpdateFile_large_repo | 30.0x faster 🚀 | 50.0x faster 🚀 |
| UpdateFile_medium_repo | 19.3x faster 🚀 | 10.2x faster 🚀 |
| UpdateFile_small_repo | 16.2x faster 🚀 | 2.2x faster 🚀 |
| UpdateFile_xlarge_repo | 105.1x faster 🚀 | 275.4x faster 🚀 |

### 💾 Memory Comparison

*Note: git-cli uses minimal memory as it stores data on disk, not in memory*

| Operation | vs git-cli | vs go-git |
|-----------|-----------|-----------|
| BulkCreateFiles_bulk_1000_files_medium | 179.2x more 💾 | 6.1x more 🔥 |
| BulkCreateFiles_bulk_1000_files_small | 1087.8x more 💾 | 23.6x more 🔥 |
| BulkCreateFiles_bulk_100_files_medium | -24.4x more 💾 | 6.3x less 💚 |
| BulkCreateFiles_bulk_100_files_small | 2.4x more 💾 | 1.5x less ✅ |
| CompareCommits_adjacent_commits_large | 58.8x more 💾 | 44.7x less 💚 |
| CompareCommits_adjacent_commits_medium | 21.0x more 💾 | 26.1x less 💚 |
| CompareCommits_adjacent_commits_small | 7.5x more 💾 | 7.0x less 💚 |
| CompareCommits_adjacent_commits_xlarge | 215.1x more 💾 | 113.6x less 💚 |
| CompareCommits_few_commits_large | 52.2x more 💾 | 56.5x less 💚 |
| CompareCommits_few_commits_medium | 26.0x more 💾 | 21.3x less 💚 |
| CompareCommits_few_commits_small | 4.1x more 💾 | 10.8x less 💚 |
| CompareCommits_few_commits_xlarge | 204.7x more 💾 | 109.0x less 💚 |
| CompareCommits_max_commits_large | 43.6x more 💾 | 66.6x less 💚 |
| CompareCommits_max_commits_medium | 15.6x more 💾 | 35.9x less 💚 |
| CompareCommits_max_commits_small | 8.0x more 💾 | 5.4x less 💚 |
| CompareCommits_max_commits_xlarge | 211.7x more 💾 | 105.1x less 💚 |
| CreateFile_large_repo | 30.8x more 💾 | 51.2x less 💚 |
| CreateFile_medium_repo | 17.0x more 💾 | 19.3x less 💚 |
| CreateFile_small_repo | 12.1x more 💾 | 3.0x less 💚 |
| CreateFile_xlarge_repo | 79.2x more 💾 | 195.0x less 💚 |
| DeleteFile_large_repo | 28.9x more 💾 | 64.4x less 💚 |
| DeleteFile_medium_repo | 15.1x more 💾 | 21.1x less 💚 |
| DeleteFile_small_repo | 8.6x more 💾 | 2.9x less 💚 |
| DeleteFile_xlarge_repo | 73.6x more 💾 | 211.8x less 💚 |
| GetFlatTree_large_tree | 0.6x more 💾 | 104.1x less 💚 |
| GetFlatTree_medium_tree | 1.4x more 💾 | 47.1x less 💚 |
| GetFlatTree_small_tree | 0.2x more 💾 | 93.0x less 💚 |
| GetFlatTree_xlarge_tree | 0.7x more 💾 | 136.8x less 💚 |
| UpdateFile_large_repo | 31.0x more 💾 | 51.5x less 💚 |
| UpdateFile_medium_repo | 21.1x more 💾 | 17.5x less 💚 |
| UpdateFile_small_repo | 14.4x more 💾 | 1.7x less ✅ |
| UpdateFile_xlarge_repo | 87.8x more 💾 | 179.7x less 💚 |

## 📈 Detailed Statistics

### BulkCreateFiles_bulk_1000_files_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 10.39s | 10.39s | 1.0 MB | 1.0 MB |
| go-git | 1 | ✅ 100.0% | 70.99s | 70.99s | 29.3 MB | 29.3 MB |
| nanogit | 1 | ✅ 100.0% | 798.7ms | 798.7ms | 179.3 MB | 179.3 MB |

### BulkCreateFiles_bulk_1000_files_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 9.95s | 9.95s | 121.3 KB | 121.3 KB |
| go-git | 1 | ✅ 100.0% | 18.18s | 18.18s | 5.5 MB | 5.5 MB |
| nanogit | 1 | ✅ 100.0% | 805.5ms | 805.5ms | 128.8 MB | 128.8 MB |

### BulkCreateFiles_bulk_100_files_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.66s | 1.66s | -226264 B | -226264 B |
| go-git | 1 | ✅ 100.0% | 6.56s | 6.56s | 33.4 MB | 33.4 MB |
| nanogit | 1 | ✅ 100.0% | 105.2ms | 105.2ms | 5.3 MB | 5.3 MB |

### BulkCreateFiles_bulk_100_files_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.29s | 1.29s | 1.3 MB | 1.3 MB |
| go-git | 1 | ✅ 100.0% | 901.0ms | 901.0ms | 4.8 MB | 4.8 MB |
| nanogit | 1 | ✅ 100.0% | 97.1ms | 97.1ms | 3.2 MB | 3.2 MB |

### CompareCommits_adjacent_commits_large

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.38s | 1.38s | 72.8 KB | 72.8 KB |
| go-git | 1 | ⚠️ 0.0% | 2.63s | 2.63s | 186.7 MB | 186.7 MB |
| nanogit | 1 | ✅ 100.0% | 124.4ms | 124.4ms | 4.2 MB | 4.2 MB |

### CompareCommits_adjacent_commits_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 562.2ms | 562.2ms | 72.4 KB | 72.4 KB |
| go-git | 1 | ⚠️ 0.0% | 416.4ms | 416.4ms | 38.6 MB | 38.6 MB |
| nanogit | 1 | ✅ 100.0% | 102.1ms | 102.1ms | 1.5 MB | 1.5 MB |

### CompareCommits_adjacent_commits_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 348.7ms | 348.7ms | 72.8 KB | 72.8 KB |
| go-git | 1 | ⚠️ 0.0% | 69.5ms | 69.5ms | 3.7 MB | 3.7 MB |
| nanogit | 1 | ✅ 100.0% | 82.2ms | 82.2ms | 544.3 KB | 544.3 KB |

### CompareCommits_adjacent_commits_xlarge

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 5.60s | 5.60s | 73.7 KB | 73.7 KB |
| go-git | 1 | ⚠️ 0.0% | 20.45s | 20.45s | 1.7 GB | 1.7 GB |
| nanogit | 1 | ✅ 100.0% | 185.1ms | 185.1ms | 15.5 MB | 15.5 MB |

### CompareCommits_few_commits_large

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.50s | 1.50s | 72.8 KB | 72.8 KB |
| go-git | 1 | ⚠️ 0.0% | 2.62s | 2.62s | 209.3 MB | 209.3 MB |
| nanogit | 1 | ✅ 100.0% | 221.7ms | 221.7ms | 3.7 MB | 3.7 MB |

### CompareCommits_few_commits_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 516.4ms | 516.4ms | 72.8 KB | 72.8 KB |
| go-git | 1 | ⚠️ 0.0% | 418.9ms | 418.9ms | 39.3 MB | 39.3 MB |
| nanogit | 1 | ✅ 100.0% | 183.6ms | 183.6ms | 1.8 MB | 1.8 MB |

### CompareCommits_few_commits_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 351.1ms | 351.1ms | 72.8 KB | 72.8 KB |
| go-git | 1 | ⚠️ 0.0% | 71.4ms | 71.4ms | 3.1 MB | 3.1 MB |
| nanogit | 1 | ✅ 100.0% | 158.6ms | 158.6ms | 296.9 KB | 296.9 KB |

### CompareCommits_few_commits_xlarge

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 5.71s | 5.71s | 72.8 KB | 72.8 KB |
| go-git | 1 | ⚠️ 0.0% | 20.51s | 20.51s | 1.5 GB | 1.5 GB |
| nanogit | 1 | ✅ 100.0% | 324.4ms | 324.4ms | 14.5 MB | 14.5 MB |

### CompareCommits_max_commits_large

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.37s | 1.37s | 73.6 KB | 73.6 KB |
| go-git | 1 | ⚠️ 0.0% | 2.61s | 2.61s | 209.2 MB | 209.2 MB |
| nanogit | 1 | ✅ 100.0% | 324.7ms | 324.7ms | 3.1 MB | 3.1 MB |

### CompareCommits_max_commits_medium

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 480.7ms | 480.7ms | 73.2 KB | 73.2 KB |
| go-git | 1 | ⚠️ 0.0% | 413.5ms | 413.5ms | 40.1 MB | 40.1 MB |
| nanogit | 1 | ✅ 100.0% | 252.7ms | 252.7ms | 1.1 MB | 1.1 MB |

### CompareCommits_max_commits_small

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 348.0ms | 348.0ms | 72.4 KB | 72.4 KB |
| go-git | 1 | ⚠️ 0.0% | 74.1ms | 74.1ms | 3.1 MB | 3.1 MB |
| nanogit | 1 | ✅ 100.0% | 226.5ms | 226.5ms | 581.4 KB | 581.4 KB |

### CompareCommits_max_commits_xlarge

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 7.88s | 7.88s | 72.6 KB | 72.6 KB |
| go-git | 1 | ⚠️ 0.0% | 20.91s | 20.91s | 1.5 GB | 1.5 GB |
| nanogit | 1 | ✅ 100.0% | 520.5ms | 520.5ms | 15.0 MB | 15.0 MB |

### CreateFile_large_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.76s | 1.79s | 139.4 KB | 139.4 KB |
| go-git | 3 | ✅ 100.0% | 2.94s | 2.95s | 214.9 MB | 178.1 MB |
| nanogit | 3 | ✅ 100.0% | 58.8ms | 63.0ms | 4.2 MB | 4.2 MB |

### CreateFile_medium_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 967.7ms | 974.7ms | 139.6 KB | 139.4 KB |
| go-git | 3 | ✅ 100.0% | 513.4ms | 518.9ms | 44.6 MB | 50.9 MB |
| nanogit | 3 | ✅ 100.0% | 51.0ms | 53.4ms | 2.3 MB | 2.8 MB |

### CreateFile_small_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 819.2ms | 889.9ms | 139.3 KB | 139.4 KB |
| go-git | 3 | ✅ 100.0% | 111.9ms | 128.4ms | 5.0 MB | 5.0 MB |
| nanogit | 3 | ✅ 100.0% | 51.1ms | 55.1ms | 1.6 MB | 1.7 MB |

### CreateFile_xlarge_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 8.82s | 11.28s | 139.7 KB | 139.4 KB |
| go-git | 3 | ✅ 100.0% | 22.65s | 22.70s | 2.1 GB | 2.0 GB |
| nanogit | 3 | ✅ 100.0% | 86.4ms | 89.4ms | 10.8 MB | 9.7 MB |

### DeleteFile_large_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.75s | 1.83s | 139.4 KB | 139.3 KB |
| go-git | 3 | ✅ 100.0% | 2.95s | 2.98s | 253.4 MB | 291.4 MB |
| nanogit | 3 | ✅ 100.0% | 58.9ms | 60.0ms | 3.9 MB | 4.2 MB |

### DeleteFile_medium_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 965.4ms | 972.6ms | 139.3 KB | 139.3 KB |
| go-git | 3 | ✅ 100.0% | 505.5ms | 520.0ms | 43.3 MB | 50.1 MB |
| nanogit | 3 | ✅ 100.0% | 50.5ms | 55.4ms | 2.1 MB | 2.1 MB |

### DeleteFile_small_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 790.2ms | 798.4ms | 139.9 KB | 139.3 KB |
| go-git | 3 | ✅ 100.0% | 105.0ms | 108.0ms | 3.4 MB | 3.7 MB |
| nanogit | 3 | ✅ 100.0% | 46.7ms | 49.6ms | 1.2 MB | 936.6 KB |

### DeleteFile_xlarge_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 6.86s | 8.28s | 139.4 KB | 139.3 KB |
| go-git | 3 | ✅ 100.0% | 22.56s | 22.64s | 2.1 GB | 2.1 GB |
| nanogit | 3 | ✅ 100.0% | 83.9ms | 93.3ms | 10.0 MB | 9.0 MB |

### GetFlatTree_large_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 1.14s | 1.14s | 3.2 MB | 3.2 MB |
| go-git | 1 | ✅ 100.0% | 2.79s | 2.79s | 206.7 MB | 206.7 MB |
| nanogit | 1 | ✅ 100.0% | 39.8ms | 39.8ms | 2.0 MB | 2.0 MB |

### GetFlatTree_medium_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 506.0ms | 506.0ms | 743.1 KB | 743.1 KB |
| go-git | 1 | ✅ 100.0% | 455.5ms | 455.5ms | 47.9 MB | 47.9 MB |
| nanogit | 1 | ✅ 100.0% | 34.5ms | 34.5ms | 1.0 MB | 1.0 MB |

### GetFlatTree_small_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 395.7ms | 395.7ms | 157.6 KB | 157.6 KB |
| go-git | 1 | ✅ 100.0% | 73.2ms | 73.2ms | 3.2 MB | 3.2 MB |
| nanogit | 1 | ✅ 100.0% | 44.6ms | 44.6ms | 35.5 KB | 35.5 KB |

### GetFlatTree_xlarge_tree

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 1 | ✅ 100.0% | 5.21s | 5.21s | 18.7 MB | 18.7 MB |
| go-git | 1 | ✅ 100.0% | 20.47s | 20.47s | 1.7 GB | 1.7 GB |
| nanogit | 1 | ✅ 100.0% | 67.6ms | 67.6ms | 12.7 MB | 12.7 MB |

### UpdateFile_large_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 1.77s | 1.93s | 138.8 KB | 138.6 KB |
| go-git | 3 | ✅ 100.0% | 2.96s | 2.98s | 216.1 MB | 181.4 MB |
| nanogit | 3 | ✅ 100.0% | 59.1ms | 62.1ms | 4.2 MB | 4.2 MB |

### UpdateFile_medium_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 950.8ms | 958.9ms | 138.6 KB | 138.6 KB |
| go-git | 3 | ✅ 100.0% | 503.1ms | 515.0ms | 50.1 MB | 50.3 MB |
| nanogit | 3 | ✅ 100.0% | 49.2ms | 51.6ms | 2.9 MB | 2.9 MB |

### UpdateFile_small_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 789.0ms | 793.8ms | 138.8 KB | 138.9 KB |
| go-git | 3 | ✅ 100.0% | 107.6ms | 113.2ms | 3.4 MB | 3.2 MB |
| nanogit | 3 | ✅ 100.0% | 48.7ms | 48.8ms | 1.9 MB | 2.5 MB |

### UpdateFile_xlarge_repo

| Client | Runs | Success | Avg Duration | P95 Duration | Avg Memory | Median Memory |
|--------|------|---------|--------------|--------------|------------|---------------|
| git-cli | 3 | ✅ 100.0% | 8.59s | 9.05s | 138.5 KB | 138.6 KB |
| go-git | 3 | ✅ 100.0% | 22.52s | 22.53s | 2.1 GB | 2.1 GB |
| nanogit | 3 | ✅ 100.0% | 81.8ms | 85.0ms | 11.9 MB | 12.9 MB |

