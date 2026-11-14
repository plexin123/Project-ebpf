// order-processing pipeline used as a profiler target
//
// what it exercises:
// - deep stacks:     main -> process_batch -> compute_price -> apply_discount -> ...
// - recursion:       apply_discount calls itself (0..3 levels)
// - shared callee:   checksum is called from validate_order and save_order
// - branches:        invalid orders skip compute_price / save_order
// - concurrency:     WORKERS threads run the same functions at the same time
// - regression:      flip SLOW_LOOKUP, commit, rebuild -> find_customer gets slower

use std::hint::black_box;
use std::thread;

// set to true, commit and rebuild to simulate a regression in find_customer
const SLOW_LOOKUP: bool = false;

const WORKERS: u64 = 3;
const ORDERS_PER_WORKER: u64 = 20;
const CUSTOMERS: u64 = 5_000;

// burn CPU inline, so the work counts as self-time of the calling function
// instead of showing up as an extra frame
macro_rules! spin {
    ($n:expr) => {
        let mut acc: u64 = 0;
        for i in 0..$n {
            acc = acc.wrapping_add(black_box(i));
        }
        black_box(acc);
    };
}

struct Order {
    id: u64,
    customer_id: u64,
    amount: u64,
}

#[inline(never)]
fn build_customers() -> Vec<u64> {
    let mut customers = Vec::with_capacity(CUSTOMERS as usize);
    for c in 0..CUSTOMERS {
        customers.push(c * 7);
    }
    customers
}

#[inline(never)]
fn checksum(bytes: &[u8]) -> u64 {
    let mut hash: u64 = 1469598103934665603;
    for b in bytes {
        hash ^= *b as u64;
        hash = hash.wrapping_mul(1099511628211);
    }
    black_box(hash)
}

#[inline(never)]
fn find_customer(customers: &[u64], customer_id: u64) -> bool {
    if SLOW_LOOKUP {
        // the "regression": someone re-sorts a copy on every lookup
        let mut copy = customers.to_vec();
        copy.sort_unstable();
        copy.binary_search(&customer_id).is_ok()
    } else {
        customers.binary_search(&customer_id).is_ok()
    }
}

#[inline(never)]
fn validate_order(order: &Order, customers: &[u64]) -> bool {
    spin!(20_000);
    let header = order.id.to_le_bytes();
    black_box(checksum(&header));
    find_customer(customers, order.customer_id)
}

#[inline(never)]
fn apply_discount(amount: u64, tier: u64) -> u64 {
    spin!(10_000);
    if tier == 0 {
        return amount;
    }
    apply_discount(black_box(amount - amount / 20), tier - 1)
}

#[inline(never)]
fn compute_tax(amount: u64) -> u64 {
    spin!(40_000);
    amount / 5
}

#[inline(never)]
fn compute_price(order: &Order) -> u64 {
    let discounted = apply_discount(order.amount, order.id % 4);
    discounted + compute_tax(discounted)
}

#[inline(never)]
fn serialize(order: &Order, price: u64) -> Vec<u8> {
    let mut buf = Vec::with_capacity(256 * 24);
    for _ in 0..256 {
        buf.extend_from_slice(&order.id.to_le_bytes());
        buf.extend_from_slice(&order.customer_id.to_le_bytes());
        buf.extend_from_slice(&price.to_le_bytes());
    }
    buf
}

#[inline(never)]
fn save_order(order: &Order, price: u64) {
    let buf = serialize(order, price);
    black_box(checksum(&buf));
}

#[inline(never)]
fn process_batch(worker: u64, customers: &[u64]) {
    for i in 0..ORDERS_PER_WORKER {
        let mut customer_id = (i * 13 % CUSTOMERS) * 7;
        // every 5th order points to a customer that does not exist
        if i % 5 == 4 {
            customer_id += 1;
        }
        let order = Order {
            id: worker * 1000 + i,
            customer_id,
            amount: 100 + i * 3,
        };
        if validate_order(&order, customers) {
            let price = compute_price(&order);
            save_order(&order, price);
        }
    }
}

fn main() {
    let customers = build_customers();
    thread::scope(|s| {
        for worker in 0..WORKERS {
            let customers = &customers;
            s.spawn(move || process_batch(worker, customers));
        }
    });
}
