# 🚀 Performance Benchmark Report

**Generated:** 2025-07-08T18:24:46+02:00  
**Total Benchmarks:** 168

## 📊 Performance Overview

| Operation                              | Speed Winner | Duration | In-Memory Winner | Memory Usage |
| -------------------------------------- | ------------ | -------- | ---------------- | ------------ |
| BulkCreateFiles_bulk_1000_files_medium | 🚀 nanogit    | 104.3ms  | 💚 nanogit        | 3.9 MB       |
| BulkCreateFiles_bulk_1000_files_small  | 🚀 nanogit    | 110.4ms  | 💚 nanogit        | 3.1 MB       |
| BulkCreateFiles_bulk_100_files_medium  | 🚀 nanogit    | 80.3ms   | 💚 nanogit        | 2.2 MB       |
| BulkCreateFiles_bulk_100_files_small   | 🚀 nanogit    | 95.6ms   | 💚 nanogit        | 1.7 MB       |
| CompareCommits_adjacent_commits_large  | 🚀 nanogit    | 112.8ms  | 💚 nanogit        | 6.3 MB       |
| CompareCommits_adjacent_commits_medium | 🚀 nanogit    | 107.2ms  | 💚 nanogit        | 2.4 MB       |
| CompareCommits_adjacent_commits_small  | 🐹 go-git     | 74.9ms   | 💚 nanogit        | 1.5 MB       |
| CompareCommits_adjacent_commits_xlarge | 🚀 nanogit    | 166.8ms  | 💚 nanogit        | 16.0 MB      |
| CompareCommits_few_commits_large       | 🚀 nanogit    | 188.8ms  | 💚 nanogit        | 6.4 MB       |
| CompareCommits_few_commits_medium      | 🚀 nanogit    | 175.3ms  | 💚 nanogit        | 3.3 MB       |
| CompareCommits_few_commits_small       | 🐹 go-git     | 89.7ms   | 💚 nanogit        | 2.1 MB       |
| CompareCommits_few_commits_xlarge      | 🚀 nanogit    | 270.5ms  | 💚 nanogit        | 16.1 MB      |
| CompareCommits_max_commits_large       | 🚀 nanogit    | 297.1ms  | 💚 nanogit        | 7.0 MB       |
| CompareCommits_max_commits_medium      | 🚀 nanogit    | 266.6ms  | 💚 nanogit        | 1.7 MB       |
| CompareCommits_max_commits_small       | 🐹 go-git     | 106.1ms  | 🐹 go-git         | 2.8 MB       |
| CompareCommits_max_commits_xlarge      | 🚀 nanogit    | 400.7ms  | 💚 nanogit        | 16.1 MB      |
| CreateFile_large_repo                  | 🚀 nanogit    | 59.6ms   | 💚 nanogit        | 4.0 MB       |
| CreateFile_medium_repo                 | 🚀 nanogit    | 51.9ms   | 💚 nanogit        | 1.5 MB       |
| CreateFile_small_repo                  | 🚀 nanogit    | 56.2ms   | 💚 nanogit        | 1.5 MB       |
| CreateFile_xlarge_repo                 | 🚀 nanogit    | 84.8ms   | 💚 nanogit        | 12.7 MB      |
| DeleteFile_large_repo                  | 🚀 nanogit    | 57.0ms   | 💚 nanogit        | 3.4 MB       |
| DeleteFile_medium_repo                 | 🚀 nanogit    | 49.9ms   | 💚 nanogit        | 1.4 MB       |
| DeleteFile_small_repo                  | 🚀 nanogit    | 47.3ms   | 💚 nanogit        | 1.4 MB       |
| DeleteFile_xlarge_repo                 | 🚀 nanogit    | 80.8ms   | 💚 nanogit        | 11.7 MB      |
| GetFlatTree_large_tree                 | 🚀 nanogit    | 58.2ms   | 💚 nanogit        | 3.6 MB       |
| GetFlatTree_medium_tree                | 🚀 nanogit    | 53.6ms   | 💚 nanogit        | 1.3 MB       |
| GetFlatTree_small_tree                 | 🚀 nanogit    | 51.1ms   | 💚 nanogit        | 697.4 KB     |
| GetFlatTree_xlarge_tree                | 🚀 nanogit    | 76.0ms   | 💚 nanogit        | 10.5 MB      |
| UpdateFile_large_repo                  | 🚀 nanogit    | 58.3ms   | 💚 nanogit        | 3.3 MB       |
| UpdateFile_medium_repo                 | 🚀 nanogit    | 50.7ms   | 💚 nanogit        | 1.4 MB       |
| UpdateFile_small_repo                  | 🚀 nanogit    | 49.1ms   | 💚 nanogit        | 1.4 MB       |
| UpdateFile_xlarge_repo                 | 🚀 nanogit    | 81.4ms   | 💚 nanogit        | 11.2 MB      |

## ⚡ Duration Comparison

| Operation                              | git-cli   | go-git    | nanogit   |
| -------------------------------------- | --------- | --------- | --------- |
| BulkCreateFiles_bulk_1000_files_medium | 10.62s 🐌  | 70.25s 🐌  | 104.3ms 🏆 |
| BulkCreateFiles_bulk_1000_files_small  | 10.18s 🐌  | 19.05s 🐌  | 110.4ms 🏆 |
| BulkCreateFiles_bulk_100_files_medium  | 1.80s 🐌   | 6.30s 🐌   | 80.3ms 🏆  |
| BulkCreateFiles_bulk_100_files_small   | 1.62s 🐌   | 815.6ms 🐌 | 95.6ms 🏆  |
| CompareCommits_adjacent_commits_large  | 1.31s 🐌   | 2.58s 🐌   | 112.8ms 🏆 |
| CompareCommits_adjacent_commits_medium | 719.5ms 🐌 | 417.4ms   | 107.2ms 🏆 |
| CompareCommits_adjacent_commits_small  | 602.5ms 🐌 | 74.9ms 🏆  | 85.0ms ✅  |
| CompareCommits_adjacent_commits_xlarge | 5.78s 🐌   | 20.23s 🐌  | 166.8ms 🏆 |
| CompareCommits_few_commits_large       | 1.36s 🐌   | 2.61s 🐌   | 188.8ms 🏆 |
| CompareCommits_few_commits_medium      | 692.2ms   | 418.6ms   | 175.3ms 🏆 |
| CompareCommits_few_commits_small       | 569.2ms 🐌 | 89.7ms 🏆  | 163.0ms ✅ |
| CompareCommits_few_commits_xlarge      | 5.58s 🐌   | 20.26s 🐌  | 270.5ms 🏆 |
| CompareCommits_max_commits_large       | 1.40s     | 2.58s 🐌   | 297.1ms 🏆 |
| CompareCommits_max_commits_medium      | 726.0ms   | 431.4ms ✅ | 266.6ms 🏆 |
| CompareCommits_max_commits_small       | 567.3ms 🐌 | 106.1ms 🏆 | 238.9ms   |
| CompareCommits_max_commits_xlarge      | 5.45s 🐌   | 20.31s 🐌  | 400.7ms 🏆 |
| CreateFile_large_repo                  | 2.24s 🐌   | 2.94s 🐌   | 59.6ms 🏆  |
| CreateFile_medium_repo                 | 1.46s 🐌   | 513.4ms 🐌 | 51.9ms 🏆  |
| CreateFile_small_repo                  | 1.31s 🐌   | 112.8ms   | 56.2ms 🏆  |
| CreateFile_xlarge_repo                 | 6.34s 🐌   | 22.39s 🐌  | 84.8ms 🏆  |
| DeleteFile_large_repo                  | 2.22s 🐌   | 2.91s 🐌   | 57.0ms 🏆  |
| DeleteFile_medium_repo                 | 1.45s 🐌   | 508.8ms 🐌 | 49.9ms 🏆  |
| DeleteFile_small_repo                  | 1.29s 🐌   | 102.9ms   | 47.3ms 🏆  |
| DeleteFile_xlarge_repo                 | 6.69s 🐌   | 22.25s 🐌  | 80.8ms 🏆  |
| GetFlatTree_large_tree                 | 1.28s 🐌   | 2.63s 🐌   | 58.2ms 🏆  |
| GetFlatTree_medium_tree                | 760.1ms 🐌 | 443.7ms 🐌 | 53.6ms 🏆  |
| GetFlatTree_small_tree                 | 594.6ms 🐌 | 78.7ms ✅  | 51.1ms 🏆  |
| GetFlatTree_xlarge_tree                | 5.23s 🐌   | 20.07s 🐌  | 76.0ms 🏆  |
| UpdateFile_large_repo                  | 2.25s 🐌   | 2.91s 🐌   | 58.3ms 🏆  |
| UpdateFile_medium_repo                 | 1.47s 🐌   | 508.9ms 🐌 | 50.7ms 🏆  |
| UpdateFile_small_repo                  | 1.31s 🐌   | 104.3ms   | 49.1ms 🏆  |
| UpdateFile_xlarge_repo                 | 6.45s 🐌   | 22.29s 🐌  | 81.4ms 🏆  |

## 💾 Memory Usage Comparison

*Note: git-cli uses disk storage rather than keeping data in memory, so memory comparisons focus on in-memory clients (nanogit vs go-git)*

| Operation                              | git-cli      | go-git     | nanogit    |
| -------------------------------------- | ------------ | ---------- | ---------- |
| BulkCreateFiles_bulk_1000_files_medium | -2643624 B 💾 | 47.2 MB 🔥  | 3.9 MB 🏆   |
| BulkCreateFiles_bulk_1000_files_small  | 322.9 KB 💾   | 9.4 MB     | 3.1 MB 🏆   |
| BulkCreateFiles_bulk_100_files_medium  | 5.1 MB 💾     | 39.4 MB 🔥  | 2.2 MB 🏆   |
| BulkCreateFiles_bulk_100_files_small   | -1018496 B 💾 | 8.0 MB     | 1.7 MB 🏆   |
| CompareCommits_adjacent_commits_large  | 70.2 KB 💾    | 226.5 MB 🔥 | 6.3 MB 🏆   |
| CompareCommits_adjacent_commits_medium | 70.5 KB 💾    | 39.2 MB 🔥  | 2.4 MB 🏆   |
| CompareCommits_adjacent_commits_small  | 70.5 KB 💾    | 6.0 MB     | 1.5 MB 🏆   |
| CompareCommits_adjacent_commits_xlarge | 70.5 KB 💾    | 1.5 GB 🔥   | 16.0 MB 🏆  |
| CompareCommits_few_commits_large       | 70.2 KB 💾    | 228.4 MB 🔥 | 6.4 MB 🏆   |
| CompareCommits_few_commits_medium      | 70.5 KB 💾    | 42.8 MB 🔥  | 3.3 MB 🏆   |
| CompareCommits_few_commits_small       | 70.5 KB 💾    | 2.5 MB ✅   | 2.1 MB 🏆   |
| CompareCommits_few_commits_xlarge      | 70.5 KB 💾    | 1.5 GB 🔥   | 16.1 MB 🏆  |
| CompareCommits_max_commits_large       | 70.5 KB 💾    | 226.8 MB 🔥 | 7.0 MB 🏆   |
| CompareCommits_max_commits_medium      | 70.5 KB 💾    | 30.3 MB 🔥  | 1.7 MB 🏆   |
| CompareCommits_max_commits_small       | 70.5 KB 💾    | 2.8 MB 🏆   | 3.0 MB ✅   |
| CompareCommits_max_commits_xlarge      | 70.2 KB 💾    | 1.6 GB 🔥   | 16.1 MB 🏆  |
| CreateFile_large_repo                  | 136.1 KB 💾   | 273.0 MB 🔥 | 4.0 MB 🏆   |
| CreateFile_medium_repo                 | 135.8 KB 💾   | 44.6 MB 🔥  | 1.5 MB 🏆   |
| CreateFile_small_repo                  | 136.2 KB 💾   | 3.6 MB     | 1.5 MB 🏆   |
| CreateFile_xlarge_repo                 | 135.9 KB 💾   | 2.0 GB 🔥   | 12.7 MB 🏆  |
| DeleteFile_large_repo                  | 136.0 KB 💾   | 282.8 MB 🔥 | 3.4 MB 🏆   |
| DeleteFile_medium_repo                 | 135.8 KB 💾   | 44.5 MB 🔥  | 1.4 MB 🏆   |
| DeleteFile_small_repo                  | 135.8 KB 💾   | 4.1 MB     | 1.4 MB 🏆   |
| DeleteFile_xlarge_repo                 | 135.8 KB 💾   | 2.0 GB 🔥   | 11.7 MB 🏆  |
| GetFlatTree_large_tree                 | 3.2 MB 💾     | 279.2 MB 🔥 | 3.6 MB 🏆   |
| GetFlatTree_medium_tree                | 740.4 KB 💾   | 39.9 MB 🔥  | 1.3 MB 🏆   |
| GetFlatTree_small_tree                 | 154.6 KB 💾   | 4.4 MB 🔥   | 697.4 KB 🏆 |
| GetFlatTree_xlarge_tree                | 18.7 MB 💾    | 1.6 GB 🔥   | 10.5 MB 🏆  |
| UpdateFile_large_repo                  | 135.3 KB 💾   | 294.6 MB 🔥 | 3.3 MB 🏆   |
| UpdateFile_medium_repo                 | 135.1 KB 💾   | 40.1 MB 🔥  | 1.4 MB 🏆   |
| UpdateFile_small_repo                  | 135.6 KB 💾   | 3.5 MB     | 1.4 MB 🏆   |
| UpdateFile_xlarge_repo                 | 135.1 KB 💾   | 2.0 GB 🔥   | 11.2 MB 🏆  |

## 🎯 Nanogit Performance Analysis

### ⚡ Speed Comparison

| Operation                              | vs git-cli      | vs go-git       |
| -------------------------------------- | --------------- | --------------- |
| BulkCreateFiles_bulk_1000_files_medium | 101.9x faster 🚀 | 673.7x faster 🚀 |
| BulkCreateFiles_bulk_1000_files_small  | 92.2x faster 🚀  | 172.6x faster 🚀 |
| BulkCreateFiles_bulk_100_files_medium  | 22.4x faster 🚀  | 78.4x faster 🚀  |
| BulkCreateFiles_bulk_100_files_small   | 16.9x faster 🚀  | 8.5x faster 🚀   |
| CompareCommits_adjacent_commits_large  | 11.6x faster 🚀  | 22.8x faster 🚀  |
| CompareCommits_adjacent_commits_medium | 6.7x faster 🚀   | 3.9x faster 🚀   |
| CompareCommits_adjacent_commits_small  | 7.1x faster 🚀   | 1.1x slower 🐌   |
| CompareCommits_adjacent_commits_xlarge | 34.6x faster 🚀  | 121.3x faster 🚀 |
| CompareCommits_few_commits_large       | 7.2x faster 🚀   | 13.8x faster 🚀  |
| CompareCommits_few_commits_medium      | 3.9x faster 🚀   | 2.4x faster 🚀   |
| CompareCommits_few_commits_small       | 3.5x faster 🚀   | 1.8x slower 🐌   |
| CompareCommits_few_commits_xlarge      | 20.6x faster 🚀  | 74.9x faster 🚀  |
| CompareCommits_max_commits_large       | 4.7x faster 🚀   | 8.7x faster 🚀   |
| CompareCommits_max_commits_medium      | 2.7x faster 🚀   | 1.6x faster ✅   |
| CompareCommits_max_commits_small       | 2.4x faster 🚀   | 2.3x slower 🐌   |
| CompareCommits_max_commits_xlarge      | 13.6x faster 🚀  | 50.7x faster 🚀  |
| CreateFile_large_repo                  | 37.6x faster 🚀  | 49.3x faster 🚀  |
| CreateFile_medium_repo                 | 28.0x faster 🚀  | 9.9x faster 🚀   |
| CreateFile_small_repo                  | 23.2x faster 🚀  | 2.0x faster 🚀   |
| CreateFile_xlarge_repo                 | 74.8x faster 🚀  | 264.0x faster 🚀 |
| DeleteFile_large_repo                  | 38.9x faster 🚀  | 51.1x faster 🚀  |
| DeleteFile_medium_repo                 | 29.1x faster 🚀  | 10.2x faster 🚀  |
| DeleteFile_small_repo                  | 27.3x faster 🚀  | 2.2x faster 🚀   |
| DeleteFile_xlarge_repo                 | 82.8x faster 🚀  | 275.4x faster 🚀 |
| GetFlatTree_large_tree                 | 22.0x faster 🚀  | 45.1x faster 🚀  |
| GetFlatTree_medium_tree                | 14.2x faster 🚀  | 8.3x faster 🚀   |
| GetFlatTree_small_tree                 | 11.6x faster 🚀  | 1.5x faster ✅   |
| GetFlatTree_xlarge_tree                | 68.8x faster 🚀  | 264.2x faster 🚀 |
| UpdateFile_large_repo                  | 38.6x faster 🚀  | 50.0x faster 🚀  |
| UpdateFile_medium_repo                 | 28.9x faster 🚀  | 10.0x faster 🚀  |
| UpdateFile_small_repo                  | 26.6x faster 🚀  | 2.1x faster 🚀   |
| UpdateFile_xlarge_repo                 | 79.2x faster 🚀  | 273.9x faster 🚀 |

### 💾 Memory Comparison

*Note: git-cli uses minimal memory as it stores data on disk, not in memory*

| Operation                              | vs git-cli    | vs go-git     |
| -------------------------------------- | ------------- | ------------- |
| BulkCreateFiles_bulk_1000_files_medium | -1.5x more 💾  | 12.3x less 💚  |
| BulkCreateFiles_bulk_1000_files_small  | 9.7x more 💾   | 3.1x less 💚   |
| BulkCreateFiles_bulk_100_files_medium  | 0.4x more 💾   | 17.6x less 💚  |
| BulkCreateFiles_bulk_100_files_small   | -1.7x more 💾  | 4.8x less 💚   |
| CompareCommits_adjacent_commits_large  | 92.1x more 💾  | 35.9x less 💚  |
| CompareCommits_adjacent_commits_medium | 35.3x more 💾  | 16.1x less 💚  |
| CompareCommits_adjacent_commits_small  | 21.2x more 💾  | 4.1x less 💚   |
| CompareCommits_adjacent_commits_xlarge | 232.4x more 💾 | 93.1x less 💚  |
| CompareCommits_few_commits_large       | 92.8x more 💾  | 35.9x less 💚  |
| CompareCommits_few_commits_medium      | 47.2x more 💾  | 13.2x less 💚  |
| CompareCommits_few_commits_small       | 29.8x more 💾  | 1.2x less ✅   |
| CompareCommits_few_commits_xlarge      | 233.3x more 💾 | 92.8x less 💚  |
| CompareCommits_max_commits_large       | 102.0x more 💾 | 32.3x less 💚  |
| CompareCommits_max_commits_medium      | 24.4x more 💾  | 18.0x less 💚  |
| CompareCommits_max_commits_small       | 43.4x more 💾  | 1.1x more ⚠️   |
| CompareCommits_max_commits_xlarge      | 234.5x more 💾 | 101.3x less 💚 |
| CreateFile_large_repo                  | 29.9x more 💾  | 68.7x less 💚  |
| CreateFile_medium_repo                 | 11.4x more 💾  | 29.5x less 💚  |
| CreateFile_small_repo                  | 11.4x more 💾  | 2.4x less 💚   |
| CreateFile_xlarge_repo                 | 95.4x more 💾  | 161.2x less 💚 |
| DeleteFile_large_repo                  | 25.5x more 💾  | 83.6x less 💚  |
| DeleteFile_medium_repo                 | 10.9x more 💾  | 30.8x less 💚  |
| DeleteFile_small_repo                  | 10.9x more 💾  | 2.9x less 💚   |
| DeleteFile_xlarge_repo                 | 88.5x more 💾  | 174.1x less 💚 |
| GetFlatTree_large_tree                 | 1.1x more 💾   | 78.5x less 💚  |
| GetFlatTree_medium_tree                | 1.8x more 💾   | 30.9x less 💚  |
| GetFlatTree_small_tree                 | 4.5x more 💾   | 6.5x less 💚   |
| GetFlatTree_xlarge_tree                | 0.6x more 💾   | 156.7x less 💚 |
| UpdateFile_large_repo                  | 24.8x more 💾  | 89.8x less 💚  |
| UpdateFile_medium_repo                 | 10.9x more 💾  | 27.8x less 💚  |
| UpdateFile_small_repo                  | 10.9x more 💾  | 2.5x less 💚   |
| UpdateFile_xlarge_repo                 | 85.0x more 💾  | 179.6x less 💚 |

## 📈 Detailed Statistics

### BulkCreateFiles_bulk_1000_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 10.62s       | 10.62s       | -2643624 B | -2643624 B    |
| go-git  | 1    | ⚠️ 0.0%   | 70.25s       | 70.25s       | 47.2 MB    | 47.2 MB       |
| nanogit | 1    | ✅ 100.0% | 104.3ms      | 104.3ms      | 3.9 MB     | 3.9 MB        |

### BulkCreateFiles_bulk_1000_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 10.18s       | 10.18s       | 322.9 KB   | 322.9 KB      |
| go-git  | 1    | ⚠️ 0.0%   | 19.05s       | 19.05s       | 9.4 MB     | 9.4 MB        |
| nanogit | 1    | ✅ 100.0% | 110.4ms      | 110.4ms      | 3.1 MB     | 3.1 MB        |

### BulkCreateFiles_bulk_100_files_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.80s        | 1.80s        | 5.1 MB     | 5.1 MB        |
| go-git  | 1    | ⚠️ 0.0%   | 6.30s        | 6.30s        | 39.4 MB    | 39.4 MB       |
| nanogit | 1    | ✅ 100.0% | 80.3ms       | 80.3ms       | 2.2 MB     | 2.2 MB        |

### BulkCreateFiles_bulk_100_files_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.62s        | 1.62s        | -1018496 B | -1018496 B    |
| go-git  | 1    | ⚠️ 0.0%   | 815.6ms      | 815.6ms      | 8.0 MB     | 8.0 MB        |
| nanogit | 1    | ✅ 100.0% | 95.6ms       | 95.6ms       | 1.7 MB     | 1.7 MB        |

### CompareCommits_adjacent_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.31s        | 1.31s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.58s        | 2.58s        | 226.5 MB   | 226.5 MB      |
| nanogit | 1    | ✅ 100.0% | 112.8ms      | 112.8ms      | 6.3 MB     | 6.3 MB        |

### CompareCommits_adjacent_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 719.5ms      | 719.5ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 417.4ms      | 417.4ms      | 39.2 MB    | 39.2 MB       |
| nanogit | 1    | ✅ 100.0% | 107.2ms      | 107.2ms      | 2.4 MB     | 2.4 MB        |

### CompareCommits_adjacent_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 602.5ms      | 602.5ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 74.9ms       | 74.9ms       | 6.0 MB     | 6.0 MB        |
| nanogit | 1    | ✅ 100.0% | 85.0ms       | 85.0ms       | 1.5 MB     | 1.5 MB        |

### CompareCommits_adjacent_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.78s        | 5.78s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.23s       | 20.23s       | 1.5 GB     | 1.5 GB        |
| nanogit | 1    | ✅ 100.0% | 166.8ms      | 166.8ms      | 16.0 MB    | 16.0 MB       |

### CompareCommits_few_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.36s        | 1.36s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.61s        | 2.61s        | 228.4 MB   | 228.4 MB      |
| nanogit | 1    | ✅ 100.0% | 188.8ms      | 188.8ms      | 6.4 MB     | 6.4 MB        |

### CompareCommits_few_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 692.2ms      | 692.2ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 418.6ms      | 418.6ms      | 42.8 MB    | 42.8 MB       |
| nanogit | 1    | ✅ 100.0% | 175.3ms      | 175.3ms      | 3.3 MB     | 3.3 MB        |

### CompareCommits_few_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 569.2ms      | 569.2ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 89.7ms       | 89.7ms       | 2.5 MB     | 2.5 MB        |
| nanogit | 1    | ✅ 100.0% | 163.0ms      | 163.0ms      | 2.1 MB     | 2.1 MB        |

### CompareCommits_few_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.58s        | 5.58s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.26s       | 20.26s       | 1.5 GB     | 1.5 GB        |
| nanogit | 1    | ✅ 100.0% | 270.5ms      | 270.5ms      | 16.1 MB    | 16.1 MB       |

### CompareCommits_max_commits_large

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.40s        | 1.40s        | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 2.58s        | 2.58s        | 226.8 MB   | 226.8 MB      |
| nanogit | 1    | ✅ 100.0% | 297.1ms      | 297.1ms      | 7.0 MB     | 7.0 MB        |

### CompareCommits_max_commits_medium

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 726.0ms      | 726.0ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 431.4ms      | 431.4ms      | 30.3 MB    | 30.3 MB       |
| nanogit | 1    | ✅ 100.0% | 266.6ms      | 266.6ms      | 1.7 MB     | 1.7 MB        |

### CompareCommits_max_commits_small

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 567.3ms      | 567.3ms      | 70.5 KB    | 70.5 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 106.1ms      | 106.1ms      | 2.8 MB     | 2.8 MB        |
| nanogit | 1    | ✅ 100.0% | 238.9ms      | 238.9ms      | 3.0 MB     | 3.0 MB        |

### CompareCommits_max_commits_xlarge

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.45s        | 5.45s        | 70.2 KB    | 70.2 KB       |
| go-git  | 1    | ⚠️ 0.0%   | 20.31s       | 20.31s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 400.7ms      | 400.7ms      | 16.1 MB    | 16.1 MB       |

### CreateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.24s        | 2.27s        | 136.1 KB   | 136.0 KB      |
| go-git  | 3    | ✅ 100.0% | 2.94s        | 3.00s        | 273.0 MB   | 274.7 MB      |
| nanogit | 3    | ✅ 100.0% | 59.6ms       | 67.2ms       | 4.0 MB     | 3.9 MB        |

### CreateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.46s        | 1.47s        | 135.8 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 513.4ms      | 528.8ms      | 44.6 MB    | 42.3 MB       |
| nanogit | 3    | ✅ 100.0% | 51.9ms       | 56.5ms       | 1.5 MB     | 1.5 MB        |

### CreateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.31s        | 1.36s        | 136.2 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 112.8ms      | 130.9ms      | 3.6 MB     | 3.9 MB        |
| nanogit | 3    | ✅ 100.0% | 56.2ms       | 77.2ms       | 1.5 MB     | 1.5 MB        |

### CreateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.34s        | 6.53s        | 135.9 KB   | 135.9 KB      |
| go-git  | 3    | ✅ 100.0% | 22.39s       | 22.44s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 84.8ms       | 94.8ms       | 12.7 MB    | 11.7 MB       |

### DeleteFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.22s        | 2.32s        | 136.0 KB   | 136.0 KB      |
| go-git  | 3    | ✅ 100.0% | 2.91s        | 2.94s        | 282.8 MB   | 280.3 MB      |
| nanogit | 3    | ✅ 100.0% | 57.0ms       | 62.4ms       | 3.4 MB     | 3.7 MB        |

### DeleteFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.45s        | 1.47s        | 135.8 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 508.8ms      | 519.1ms      | 44.5 MB    | 45.5 MB       |
| nanogit | 3    | ✅ 100.0% | 49.9ms       | 52.9ms       | 1.4 MB     | 1.4 MB        |

### DeleteFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.29s        | 1.31s        | 135.8 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 102.9ms      | 104.3ms      | 4.1 MB     | 4.8 MB        |
| nanogit | 3    | ✅ 100.0% | 47.3ms       | 52.0ms       | 1.4 MB     | 1.4 MB        |

### DeleteFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.69s        | 7.24s        | 135.8 KB   | 135.8 KB      |
| go-git  | 3    | ✅ 100.0% | 22.25s       | 22.32s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 80.8ms       | 85.7ms       | 11.7 MB    | 11.6 MB       |

### GetFlatTree_large_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 1.28s        | 1.28s        | 3.2 MB     | 3.2 MB        |
| go-git  | 1    | ✅ 100.0% | 2.63s        | 2.63s        | 279.2 MB   | 279.2 MB      |
| nanogit | 1    | ✅ 100.0% | 58.2ms       | 58.2ms       | 3.6 MB     | 3.6 MB        |

### GetFlatTree_medium_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 760.1ms      | 760.1ms      | 740.4 KB   | 740.4 KB      |
| go-git  | 1    | ✅ 100.0% | 443.7ms      | 443.7ms      | 39.9 MB    | 39.9 MB       |
| nanogit | 1    | ✅ 100.0% | 53.6ms       | 53.6ms       | 1.3 MB     | 1.3 MB        |

### GetFlatTree_small_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 594.6ms      | 594.6ms      | 154.6 KB   | 154.6 KB      |
| go-git  | 1    | ✅ 100.0% | 78.7ms       | 78.7ms       | 4.4 MB     | 4.4 MB        |
| nanogit | 1    | ✅ 100.0% | 51.1ms       | 51.1ms       | 697.4 KB   | 697.4 KB      |

### GetFlatTree_xlarge_tree

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 1    | ✅ 100.0% | 5.23s        | 5.23s        | 18.7 MB    | 18.7 MB       |
| go-git  | 1    | ✅ 100.0% | 20.07s       | 20.07s       | 1.6 GB     | 1.6 GB        |
| nanogit | 1    | ✅ 100.0% | 76.0ms       | 76.0ms       | 10.5 MB    | 10.5 MB       |

### UpdateFile_large_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 2.25s        | 2.34s        | 135.3 KB   | 135.3 KB      |
| go-git  | 3    | ✅ 100.0% | 2.91s        | 2.92s        | 294.6 MB   | 283.5 MB      |
| nanogit | 3    | ✅ 100.0% | 58.3ms       | 64.3ms       | 3.3 MB     | 3.4 MB        |

### UpdateFile_medium_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.47s        | 1.49s        | 135.1 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 508.9ms      | 514.2ms      | 40.1 MB    | 46.0 MB       |
| nanogit | 3    | ✅ 100.0% | 50.7ms       | 53.0ms       | 1.4 MB     | 1.5 MB        |

### UpdateFile_small_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 1.31s        | 1.32s        | 135.6 KB   | 135.6 KB      |
| go-git  | 3    | ✅ 100.0% | 104.3ms      | 107.6ms      | 3.5 MB     | 3.2 MB        |
| nanogit | 3    | ✅ 100.0% | 49.1ms       | 50.7ms       | 1.4 MB     | 1.4 MB        |

### UpdateFile_xlarge_repo

| Client  | Runs | Success  | Avg Duration | P95 Duration | Avg Memory | Median Memory |
| ------- | ---- | -------- | ------------ | ------------ | ---------- | ------------- |
| git-cli | 3    | ✅ 100.0% | 6.45s        | 6.84s        | 135.1 KB   | 135.1 KB      |
| go-git  | 3    | ✅ 100.0% | 22.29s       | 22.54s       | 2.0 GB     | 2.0 GB        |
| nanogit | 3    | ✅ 100.0% | 81.4ms       | 90.8ms       | 11.2 MB    | 11.0 MB       |

