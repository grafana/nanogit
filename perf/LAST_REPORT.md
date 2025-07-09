# 🚀 Performance Benchmark Report

**Generated:** 2025-07-09T10:18:29+02:00  
**Total Benchmarks:** 168

## 📊 Performance Overview

| Operation                              | Speed Winner | Duration | In-Memory Winner | Memory Usage |
| -------------------------------------- | ------------ | -------- | ---------------- | ------------ |
| BulkCreateFiles_bulk_1000_files_medium | 🚀 nanogit    | 102.8ms  | 💚 nanogit        | 3.4 MB       |
| BulkCreateFiles_bulk_1000_files_small  | 🚀 nanogit    | 83.2ms   | 💚 nanogit        | 3.4 MB       |
| BulkCreateFiles_bulk_100_files_medium  | 🚀 nanogit    | 86.3ms   | 💚 nanogit        | 2.6 MB       |
| BulkCreateFiles_bulk_100_files_small   | 🚀 nanogit    | 83.4ms   | 💚 nanogit        | 2.0 MB       |
| CompareCommits_adjacent_commits_large  | 🚀 nanogit    | 94.2ms   | 💚 nanogit        | 6.3 MB       |
| CompareCommits_adjacent_commits_medium | 🚀 nanogit    | 78.0ms   | 💚 nanogit        | 2.7 MB       |
| CompareCommits_adjacent_commits_small  | 🚀 nanogit    | 67.2ms   | 💚 nanogit        | 1.4 MB       |
| CompareCommits_adjacent_commits_xlarge | 🚀 nanogit    | 122.8ms  | 💚 nanogit        | 16.7 MB      |
| CompareCommits_few_commits_large       | 🚀 nanogit    | 170.0ms  | 💚 nanogit        | 6.3 MB       |
| CompareCommits_few_commits_medium      | 🚀 nanogit    | 156.4ms  | 💚 nanogit        | 3.4 MB       |
| CompareCommits_few_commits_small       | 🐹 go-git     | 72.6ms   | 💚 nanogit        | 2.3 MB       |
| CompareCommits_few_commits_xlarge      | 🚀 nanogit    | 206.0ms  | 💚 nanogit        | 17.1 MB      |
| CompareCommits_max_commits_large       | 🚀 nanogit    | 250.5ms  | 💚 nanogit        | 7.0 MB       |
| CompareCommits_max_commits_medium      | 🚀 nanogit    | 233.4ms  | 💚 nanogit        | 4.4 MB       |
| CompareCommits_max_commits_small       | 🐹 go-git     | 76.5ms   | 🐹 go-git         | 2.4 MB       |
| CompareCommits_max_commits_xlarge      | 🚀 nanogit    | 333.5ms  | 💚 nanogit        | 16.6 MB      |
| CreateFile_large_repo                  | 🚀 nanogit    | 60.7ms   | 💚 nanogit        | 3.3 MB       |
| CreateFile_medium_repo                 | 🚀 nanogit    | 54.0ms   | 💚 nanogit        | 2.2 MB       |
| CreateFile_small_repo                  | 🚀 nanogit    | 58.0ms   | 💚 nanogit        | 1.5 MB       |
| CreateFile_xlarge_repo                 | 🚀 nanogit    | 79.4ms   | 💚 nanogit        | 10.6 MB      |
| DeleteFile_large_repo                  | 🚀 nanogit    | 57.4ms   | 💚 nanogit        | 2.8 MB       |
| DeleteFile_medium_repo                 | 🚀 nanogit    | 50.3ms   | 💚 nanogit        | 1.2 MB       |
| DeleteFile_small_repo                  | 🚀 nanogit    | 46.4ms   | 💚 nanogit        | 1.2 MB       |
| DeleteFile_xlarge_repo                 | 🚀 nanogit    | 79.6ms   | 💚 nanogit        | 10.0 MB      |
| GetFlatTree_large_tree                 | 🚀 nanogit    | 56.5ms   | 💚 nanogit        | 3.5 MB       |
| GetFlatTree_medium_tree                | 🚀 nanogit    | 53.2ms   | 💚 nanogit        | 1.3 MB       |
| GetFlatTree_small_tree                 | 🚀 nanogit    | 52.3ms   | 💚 nanogit        | 695.3 KB     |
| GetFlatTree_xlarge_tree                | 🚀 nanogit    | 76.1ms   | 💚 nanogit        | 10.4 MB      |
| UpdateFile_large_repo                  | 🚀 nanogit    | 59.7ms   | 💚 nanogit        | 2.8 MB       |
| UpdateFile_medium_repo                 | 🚀 nanogit    | 49.7ms   | 💚 nanogit        | 1.5 MB       |
| UpdateFile_small_repo                  | 🚀 nanogit    | 48.0ms   | 💚 nanogit        | 1.4 MB       |
| UpdateFile_xlarge_repo                 | 🚀 nanogit    | 75.1ms   | 💚 nanogit        | 10.7 MB      |

## ⚡ Duration Comparison

| Operation                              | git-cli   | go-git    | nanogit   |
| -------------------------------------- | --------- | --------- | --------- |
| BulkCreateFiles_bulk_1000_files_medium | 10.70s 🐌  | 72.43s 🐌  | 102.8ms 🏆 |
| BulkCreateFiles_bulk_1000_files_small  | 9.89s 🐌   | 19.45s 🐌  | 83.2ms 🏆  |
| BulkCreateFiles_bulk_100_files_medium  | 1.70s 🐌   | 6.31s 🐌   | 86.3ms 🏆  |
| BulkCreateFiles_bulk_100_files_small   | 1.85s 🐌   | 820.5ms 🐌 | 83.4ms 🏆  |
| CompareCommits_adjacent_commits_large  | 1.52s 🐌   | 2.59s 🐌   | 94.2ms 🏆  |
| CompareCommits_adjacent_commits_medium | 757.9ms 🐌 | 411.6ms 🐌 | 78.0ms 🏆  |
| CompareCommits_adjacent_commits_small  | 581.6ms 🐌 | 67.8ms ✅  | 67.2ms 🏆  |
| CompareCommits_adjacent_commits_xlarge | 5.77s 🐌   | 20.30s 🐌  | 122.8ms 🏆 |
| CompareCommits_few_commits_large       | 1.47s 🐌   | 2.59s 🐌   | 170.0ms 🏆 |
| CompareCommits_few_commits_medium      | 839.0ms 🐌 | 407.1ms   | 156.4ms 🏆 |
| CompareCommits_few_commits_small       | 580.9ms 🐌 | 72.6ms 🏆  | 148.3ms   |
| CompareCommits_few_commits_xlarge      | 5.86s 🐌   | 20.32s 🐌  | 206.0ms 🏆 |
| CompareCommits_max_commits_large       | 1.47s 🐌   | 2.57s 🐌   | 250.5ms 🏆 |
| CompareCommits_max_commits_medium      | 769.8ms   | 423.4ms ✅ | 233.4ms 🏆 |
| CompareCommits_max_commits_small       | 588.1ms 🐌 | 76.5ms 🏆  | 225.1ms   |
| CompareCommits_max_commits_xlarge      | 5.90s 🐌   | 20.39s 🐌  | 333.5ms 🏆 |
| CreateFile_large_repo                  | 2.23s 🐌   | 2.92s 🐌   | 60.7ms 🏆  |
| CreateFile_medium_repo                 | 1.50s 🐌   | 519.5ms 🐌 | 54.0ms 🏆  |
| CreateFile_small_repo                  | 1.31s 🐌   | 114.0ms ✅ | 58.0ms 🏆  |
| CreateFile_xlarge_repo                 | 7.09s 🐌   | 22.34s 🐌  | 79.4ms 🏆  |
| DeleteFile_large_repo                  | 2.26s 🐌   | 2.92s 🐌   | 57.4ms 🏆  |
| DeleteFile_medium_repo                 | 1.51s 🐌   | 520.4ms 🐌 | 50.3ms 🏆  |
| DeleteFile_small_repo                  | 1.32s 🐌   | 99.5ms    | 46.4ms 🏆  |
| DeleteFile_xlarge_repo                 | 6.85s 🐌   | 22.34s 🐌  | 79.6ms 🏆  |
| GetFlatTree_large_tree                 | 1.42s 🐌   | 2.61s 🐌   | 56.5ms 🏆  |
| GetFlatTree_medium_tree                | 761.7ms 🐌 | 445.8ms 🐌 | 53.2ms 🏆  |
| GetFlatTree_small_tree                 | 576.4ms 🐌 | 75.1ms ✅  | 52.3ms 🏆  |
| GetFlatTree_xlarge_tree                | 6.53s 🐌   | 19.84s 🐌  | 76.1ms 🏆  |
| UpdateFile_large_repo                  | 2.21s 🐌   | 2.91s 🐌   | 59.7ms 🏆  |
| UpdateFile_medium_repo                 | 1.50s 🐌   | 517.6ms 🐌 | 49.7ms 🏆  |
| UpdateFile_small_repo                  | 1.35s 🐌   | 101.9ms   | 48.0ms 🏆  |
| UpdateFile_xlarge_repo                 | 6.97s 🐌   | 22.33s 🐌  | 75.1ms 🏆  |

## 💾 Memory Usage Comparison

*Note: git-cli uses disk storage rather than keeping data in memory, so memory comparisons focus on in-memory clients (nanogit vs go-git)*

| Operation                              | git-cli      | go-git     | nanogit    |
| -------------------------------------- | ------------ | ---------- | ---------- |
| BulkCreateFiles_bulk_1000_files_medium | -3106536 B 💾 | 31.8 MB 🔥  | 3.4 MB 🏆   |
| BulkCreateFiles_bulk_1000_files_small  | 2.9 MB 💾     | 7.3 MB     | 3.4 MB 🏆   |
| BulkCreateFiles_bulk_100_files_medium  | 5.1 MB 💾     | 31.0 MB 🔥  | 2.6 MB 🏆   |
| BulkCreateFiles_bulk_100_files_small   | -1046224 B 💾 | 8.3 MB     | 2.0 MB 🏆   |
| CompareCommits_adjacent_commits_large  | 71.4 KB 💾    | 234.9 MB 🔥 | 6.3 MB 🏆   |
| CompareCommits_adjacent_commits_medium | 70.2 KB 💾    | 45.3 MB 🔥  | 2.7 MB 🏆   |
| CompareCommits_adjacent_commits_small  | 70.5 KB 💾    | 6.1 MB     | 1.4 MB 🏆   |
| CompareCommits_adjacent_commits_xlarge | 70.2 KB 💾    | 1.4 GB 🔥   | 16.7 MB 🏆  |
| CompareCommits_few_commits_large       | 71.4 KB 💾    | 231.7 MB 🔥 | 6.3 MB 🏆   |
| CompareCommits_few_commits_medium      | 70.5 KB 💾    | 44.1 MB 🔥  | 3.4 MB 🏆   |
| CompareCommits_few_commits_small       | 70.5 KB 💾    | 2.3 MB ✅   | 2.3 MB 🏆   |
| CompareCommits_few_commits_xlarge      | 70.2 KB 💾    | 1.6 GB 🔥   | 17.1 MB 🏆  |
| CompareCommits_max_commits_large       | 70.2 KB 💾    | 229.0 MB 🔥 | 7.0 MB 🏆   |
| CompareCommits_max_commits_medium      | 70.5 KB 💾    | 35.3 MB 🔥  | 4.4 MB 🏆   |
| CompareCommits_max_commits_small       | 70.2 KB 💾    | 2.4 MB 🏆   | 3.2 MB ✅   |
| CompareCommits_max_commits_xlarge      | 70.5 KB 💾    | 1.6 GB 🔥   | 16.6 MB 🏆  |
| CreateFile_large_repo                  | 135.8 KB 💾   | 278.7 MB 🔥 | 3.3 MB 🏆   |
| CreateFile_medium_repo                 | 136.2 KB 💾   | 38.8 MB 🔥  | 2.2 MB 🏆   |
| CreateFile_small_repo                  | 136.7 KB 💾   | 3.0 MB     | 1.5 MB 🏆   |
| CreateFile_xlarge_repo                 | 135.8 KB 💾   | 2.0 GB 🔥   | 10.6 MB 🏆  |
| DeleteFile_large_repo                  | 135.6 KB 💾   | 277.3 MB 🔥 | 2.8 MB 🏆   |
| DeleteFile_medium_repo                 | 135.8 KB 💾   | 34.6 MB 🔥  | 1.2 MB 🏆   |
| DeleteFile_small_repo                  | 135.8 KB 💾   | 3.4 MB     | 1.2 MB 🏆   |
| DeleteFile_xlarge_repo                 | 135.8 KB 💾   | 2.0 GB 🔥   | 10.0 MB 🏆  |
| GetFlatTree_large_tree                 | 3.2 MB 💾     | 245.7 MB 🔥 | 3.5 MB 🏆   |
| GetFlatTree_medium_tree                | 740.1 KB 💾   | 31.9 MB 🔥  | 1.3 MB 🏆   |
| GetFlatTree_small_tree                 | 154.9 KB 💾   | 4.4 MB 🔥   | 695.3 KB 🏆 |
| GetFlatTree_xlarge_tree                | 18.7 MB 💾    | 1.6 GB 🔥   | 10.4 MB 🏆  |
| UpdateFile_large_repo                  | 135.0 KB 💾   | 281.5 MB 🔥 | 2.8 MB 🏆   |
| UpdateFile_medium_repo                 | 136.2 KB 💾   | 29.7 MB 🔥  | 1.5 MB 🏆   |
| UpdateFile_small_repo                  | 135.2 KB 💾   | 4.5 MB     | 1.4 MB 🏆   |
| UpdateFile_xlarge_repo                 | 135.1 KB 💾   | 2.0 GB 🔥   | 10.7 MB 🏆  |

## 🎯 Nanogit Performance Analysis

### ⚡ Speed Comparison

| Operation                              | vs git-cli      | vs go-git       |
| -------------------------------------- | --------------- | --------------- |
| BulkCreateFiles_bulk_1000_files_medium | 104.1x faster 🚀 | 704.3x faster 🚀 |
| BulkCreateFiles_bulk_1000_files_small  | 118.8x faster 🚀 | 233.6x faster 🚀 |
| BulkCreateFiles_bulk_100_files_medium  | 19.8x faster 🚀  | 73.1x faster 🚀  |
| BulkCreateFiles_bulk_100_files_small   | 22.2x faster 🚀  | 9.8x faster 🚀   |
| CompareCommits_adjacent_commits_large  | 16.2x faster 🚀  | 27.4x faster 🚀  |
| CompareCommits_adjacent_commits_medium | 9.7x faster 🚀   | 5.3x faster 🚀   |
| CompareCommits_adjacent_commits_small  | 8.7x faster 🚀   | ~same ⚖️         |
| CompareCommits_adjacent_commits_xlarge | 47.0x faster 🚀  | 165.3x faster 🚀 |
| CompareCommits_few_commits_large       | 8.7x faster 🚀   | 15.2x faster 🚀  |
| CompareCommits_few_commits_medium      | 5.4x faster 🚀   | 2.6x faster 🚀   |
| CompareCommits_few_commits_small       | 3.9x faster 🚀   | 2.0x slower 🐌   |
| CompareCommits_few_commits_xlarge      | 28.5x faster 🚀  | 98.6x faster 🚀  |
| CompareCommits_max_commits_large       | 5.9x faster 🚀   | 10.3x faster 🚀  |
| CompareCommits_max_commits_medium      | 3.3x faster 🚀   | 1.8x faster ✅   |
| CompareCommits_max_commits_small       | 2.6x faster 🚀   | 2.9x slower 🐌   |
| CompareCommits_max_commits_xlarge      | 17.7x faster 🚀  | 61.1x faster 🚀  |
| CreateFile_large_repo                  | 36.7x faster 🚀  | 48.1x faster 🚀  |
| CreateFile_medium_repo                 | 27.7x faster 🚀  | 9.6x faster 🚀   |
| CreateFile_small_repo                  | 22.5x faster 🚀  | 2.0x faster ✅   |
| CreateFile_xlarge_repo                 | 89.4x faster 🚀  | 281.6x faster 🚀 |
| DeleteFile_large_repo                  | 39.3x faster 🚀  | 50.9x faster 🚀  |
| DeleteFile_medium_repo                 | 30.1x faster 🚀  | 10.4x faster 🚀  |
| DeleteFile_small_repo                  | 28.4x faster 🚀  | 2.1x faster 🚀   |
| DeleteFile_xlarge_repo                 | 86.0x faster 🚀  | 280.5x faster 🚀 |
| GetFlatTree_large_tree                 | 25.1x faster 🚀  | 46.1x faster 🚀  |
| GetFlatTree_medium_tree                | 14.3x faster 🚀  | 8.4x faster 🚀   |
| GetFlatTree_small_tree                 | 11.0x faster 🚀  | 1.4x faster ✅   |
| GetFlatTree_xlarge_tree                | 85.8x faster 🚀  | 260.8x faster 🚀 |
| UpdateFile_large_repo                  | 37.1x faster 🚀  | 48.7x faster 🚀  |
| UpdateFile_medium_repo                 | 30.2x faster 🚀  | 10.4x faster 🚀  |
| UpdateFile_small_repo                  | 28.2x faster 🚀  | 2.1x faster 🚀   |
| UpdateFile_xlarge_repo                 | 92.8x faster 🚀  | 297.3x faster 🚀 |

### 💾 Memory Comparison

*Note: git-cli uses minimal memory as it stores data on disk, not in memory*

| Operation                              | vs git-cli    | vs go-git     |
| -------------------------------------- | ------------- | ------------- |
| BulkCreateFiles_bulk_1000_files_medium | -1.2x more 💾  | 9.3x less 💚   |
| BulkCreateFiles_bulk_1000_files_small  | 1.2x more 💾   | 2.1x less 💚   |
| BulkCreateFiles_bulk_100_files_medium  | 0.5x more 💾   | 11.7x less 💚  |
| BulkCreateFiles_bulk_100_files_small   | -2.1x more 💾  | 4.1x less 💚   |
| CompareCommits_adjacent_commits_large  | 90.5x more 💾  | 37.2x less 💚  |
| CompareCommits_adjacent_commits_medium | 39.3x more 💾  | 16.8x less 💚  |
| CompareCommits_adjacent_commits_small  | 20.6x more 💾  | 4.3x less 💚   |
| CompareCommits_adjacent_commits_xlarge | 243.1x more 💾 | 88.3x less 💚  |
| CompareCommits_few_commits_large       | 90.7x more 💾  | 36.6x less 💚  |
| CompareCommits_few_commits_medium      | 49.8x more 💾  | 12.8x less 💚  |
| CompareCommits_few_commits_small       | 33.5x more 💾  | 1.0x less ✅   |
| CompareCommits_few_commits_xlarge      | 249.5x more 💾 | 95.6x less 💚  |
| CompareCommits_max_commits_large       | 101.6x more 💾 | 32.9x less 💚  |
| CompareCommits_max_commits_medium      | 64.4x more 💾  | 8.0x less 💚   |
| CompareCommits_max_commits_small       | 46.1x more 💾  | 1.3x more ⚠️   |
| CompareCommits_max_commits_xlarge      | 240.8x more 💾 | 98.1x less 💚  |
| CreateFile_large_repo                  | 25.2x more 💾  | 83.3x less 💚  |
| CreateFile_medium_repo                 | 16.5x more 💾  | 17.7x less 💚  |
| CreateFile_small_repo                  | 11.2x more 💾  | 2.0x less 💚   |
| CreateFile_xlarge_repo                 | 79.7x more 💾  | 198.4x less 💚 |
| DeleteFile_large_repo                  | 21.5x more 💾  | 97.6x less 💚  |
| DeleteFile_medium_repo                 | 8.8x more 💾   | 29.5x less 💚  |
| DeleteFile_small_repo                  | 8.9x more 💾   | 2.9x less 💚   |
| DeleteFile_xlarge_repo                 | 75.6x more 💾  | 200.5x less 💚 |
| GetFlatTree_large_tree                 | 1.1x more 💾   | 69.3x less 💚  |
| GetFlatTree_medium_tree                | 1.8x more 💾   | 24.8x less 💚  |
| GetFlatTree_small_tree                 | 4.5x more 💾   | 6.5x less 💚   |
| GetFlatTree_xlarge_tree                | 0.6x more 💾   | 154.3x less 💚 |
| UpdateFile_large_repo                  | 21.3x more 💾  | 100.2x less 💚 |
| UpdateFile_medium_repo                 | 11.1x more 💾  | 20.1x less 💚  |
| UpdateFile_small_repo                  | 10.8x more 💾  | 3.1x less 💚   |
| UpdateFile_xlarge_repo                 | 81.1x more 💾  | 189.2x less 💚 |

## 📈 Detailed Statistics

### BulkCreateFiles_bulk_1000_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 10.70s       | 10.70s       | -3106536 B | -3106536 B    |
| go-git  | 1    | ⚠️ 0.0%   | 72.43s       | 72.43s       | 31.8 MB    | 31.8 MB       |
| nanogit | 1    | ✅ 100.0% | 102.8ms      | 102.8ms      | 3.4 MB     | 3.4 MB        |

### BulkCreateFiles_bulk_1000_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 9.89s        | 9.89s        | 2.9 MB     | 2.9 MB        |
| go-git  | 1    | ⚠️ 0.0%   | 19.45s       | 19.45s       | 7.3 MB     | 7.3 MB        |
| nanogit | 1    | ✅ 100.0% | 83.2ms       | 83.2ms       | 3.4 MB     | 3.4 MB        |

### BulkCreateFiles_bulk_100_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.70s        | 1.70s        | 5.1 MB     | 5.1 MB        |
| go-git  | 1    | ⚠️ 0.0%   | 6.31s        | 6.31s        | 31.0 MB    | 31.0 MB       |
| nanogit | 1    | ✅ 100.0% | 86.3ms       | 86.3ms       | 2.6 MB     | 2.6 MB        |

### BulkCreateFiles_bulk_100_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.85s        | 1.85s        | -1046224 B | -1046224 B    |
| go-git  | 1    | ⚠️ 0.0%   | 820.5ms      | 820.5ms      | 8.3 MB     | 8.3 MB        |
| nanogit | 1    | ✅ 100.0% | 83.4ms       | 83.4ms       | 2.0 MB     | 2.0 MB        |

### CompareCommits_adjacent_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.52s        | 1.52s        | 71.4 KB    | 71.4 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.59s        | 2.59s        | 234.9 MB   | 234.9 MB      |
| nanogit | 1    | ✅ 100.0% | 94.2ms       | 94.2ms       | 6.3 MB     | 6.3 MB        |

### CompareCommits_adjacent_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 757.9ms      | 757.9ms      | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 411.6ms      | 411.6ms      | 45.3 MB    | 45.3 MB       |
| nanogit | 1    | ✅ 100.0% | 78.0ms       | 78.0ms       | 2.7 MB     | 2.7 MB        |

### CompareCommits_adjacent_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 581.6ms      | 581.6ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 67.8ms       | 67.8ms       | 6.1 MB     | 6.1 MB        |
| nanogit | 1    | ✅ 100.0% | 67.2ms       | 67.2ms       | 1.4 MB     | 1.4 MB        |

### CompareCommits_adjacent_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.77s        | 5.77s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.30s       | 20.30s       | 1.4 GB     | 1.4 GB        |
| nanogit | 1    | ✅ 100.0% | 122.8ms      | 122.8ms      | 16.7 MB    | 16.7 MB       |

### CompareCommits_few_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.47s        | 1.47s        | 71.4 KB    | 71.4 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.59s        | 2.59s        | 231.7 MB   | 231.7 MB      |
| nanogit | 1    | ✅ 100.0% | 170.0ms      | 170.0ms      | 6.3 MB     | 6.3 MB        |

### CompareCommits_few_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 839.0ms      | 839.0ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 407.1ms      | 407.1ms      | 44.1 MB    | 44.1 MB       |
| nanogit | 1    | ✅ 100.0% | 156.4ms      | 156.4ms      | 3.4 MB     | 3.4 MB        |

### CompareCommits_few_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 580.9ms      | 580.9ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 72.6ms       | 72.6ms       | 2.3 MB     | 2.3 MB        |
| nanogit | 1    | ✅ 100.0% | 148.3ms      | 148.3ms      | 2.3 MB     | 2.3 MB        |

### CompareCommits_few_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.86s        | 5.86s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.32s       | 20.32s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 206.0ms      | 206.0ms      | 17.1 MB    | 17.1 MB       |

### CompareCommits_max_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.47s        | 1.47s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.57s        | 2.57s        | 229.0 MB   | 229.0 MB      |
| nanogit | 1    | ✅ 100.0% | 250.5ms      | 250.5ms      | 7.0 MB     | 7.0 MB        |

### CompareCommits_max_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 769.8ms      | 769.8ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 423.4ms      | 423.4ms      | 35.3 MB    | 35.3 MB       |
| nanogit | 1    | ✅ 100.0% | 233.4ms      | 233.4ms      | 4.4 MB     | 4.4 MB        |

### CompareCommits_max_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 588.1ms      | 588.1ms      | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 76.5ms       | 76.5ms       | 2.4 MB     | 2.4 MB        |
| nanogit | 1    | ✅ 100.0% | 225.1ms      | 225.1ms      | 3.2 MB     | 3.2 MB        |

### CompareCommits_max_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.90s        | 5.90s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.39s       | 20.39s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 333.5ms      | 333.5ms      | 16.6 MB    | 16.6 MB       |

### CreateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.23s        | 2.32s        | 135.8 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 2.92s        | 2.95s        | 278.7 MB   | 280.2 MB      |
| nanogit | 3    | ✅ 100.0% | 60.7ms       | 63.1ms       | 3.3 MB     | 2.8 MB        |

### CreateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.50s        | 1.51s        | 136.2 KB   | 136.2 KB      |
| go-git  | 3    | ✅ 100.0% | 519.5ms      | 541.3ms      | 38.8 MB    | 33.1 MB       |
| nanogit | 3    | ✅ 100.0% | 54.0ms       | 56.6ms       | 2.2 MB     | 2.2 MB        |

### CreateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.31s        | 1.32s        | 136.7 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 114.0ms      | 146.7ms      | 3.0 MB     | 2.8 MB        |
| nanogit | 3    | ✅ 100.0% | 58.0ms       | 79.6ms       | 1.5 MB     | 1.6 MB        |

### CreateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 7.09s        | 8.43s        | 135.8 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 22.34s       | 22.53s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 79.4ms       | 83.0ms       | 10.6 MB    | 10.3 MB       |

### DeleteFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.26s        | 2.27s        | 135.6 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 2.92s        | 2.93s        | 277.3 MB   | 274.8 MB      |
| nanogit | 3    | ✅ 100.0% | 57.4ms       | 61.3ms       | 2.8 MB     | 2.8 MB        |

### DeleteFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.51s        | 1.54s        | 135.8 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 520.4ms      | 530.3ms      | 34.6 MB    | 39.5 MB       |
| nanogit | 3    | ✅ 100.0% | 50.3ms       | 50.9ms       | 1.2 MB     | 1.3 MB        |

### DeleteFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.32s        | 1.34s        | 135.8 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 99.5ms       | 100.4ms      | 3.4 MB     | 3.4 MB        |
| nanogit | 3    | ✅ 100.0% | 46.4ms       | 49.0ms       | 1.2 MB     | 862.9 KB      |

### DeleteFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.85s        | 7.15s        | 135.8 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 22.34s       | 22.47s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 79.6ms       | 83.9ms       | 10.0 MB    | 10.4 MB       |

### GetFlatTree_large_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.42s        | 1.42s        | 3.2 MB     | 3.2 MB        |
| go-git  | 1    | ✅ 100.0% | 2.61s        | 2.61s        | 245.7 MB   | 245.7 MB      |
| nanogit | 1    | ✅ 100.0% | 56.5ms       | 56.5ms       | 3.5 MB     | 3.5 MB        |

### GetFlatTree_medium_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 761.7ms      | 761.7ms      | 740.1 KB   | 740.1 KB      |
| go-git  | 1    | ✅ 100.0% | 445.8ms      | 445.8ms      | 31.9 MB    | 31.9 MB       |
| nanogit | 1    | ✅ 100.0% | 53.2ms       | 53.2ms       | 1.3 MB     | 1.3 MB        |

### GetFlatTree_small_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 576.4ms      | 576.4ms      | 154.9 KB   | 154.9 KB      |
| go-git  | 1    | ✅ 100.0% | 75.1ms       | 75.1ms       | 4.4 MB     | 4.4 MB        |
| nanogit | 1    | ✅ 100.0% | 52.3ms       | 52.3ms       | 695.3 KB   | 695.3 KB      |

### GetFlatTree_xlarge_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 6.53s        | 6.53s        | 18.7 MB    | 18.7 MB       |
| go-git  | 1    | ✅ 100.0% | 19.84s       | 19.84s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 76.1ms       | 76.1ms       | 10.4 MB    | 10.4 MB       |

### UpdateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.21s        | 2.28s        | 135.0 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 2.91s        | 2.92s        | 281.5 MB   | 281.4 MB      |
| nanogit | 3    | ✅ 100.0% | 59.7ms       | 63.5ms       | 2.8 MB     | 2.9 MB        |

### UpdateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.50s        | 1.52s        | 136.2 KB   | 136.2 KB      |
| go-git  | 3    | ✅ 100.0% | 517.6ms      | 531.7ms      | 29.7 MB    | 26.4 MB       |
| nanogit | 3    | ✅ 100.0% | 49.7ms       | 53.3ms       | 1.5 MB     | 1.5 MB        |

### UpdateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.35s        | 1.42s        | 135.2 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 101.9ms      | 106.5ms      | 4.5 MB     | 4.3 MB        |
| nanogit | 3    | ✅ 100.0% | 48.0ms       | 51.9ms       | 1.4 MB     | 1.3 MB        |

### UpdateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.97s        | 7.21s        | 135.1 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 22.33s       | 22.44s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 75.1ms       | 76.9ms       | 10.7 MB    | 10.4 MB       |

