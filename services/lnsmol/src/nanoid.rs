use rand::distributions::Distribution;

/// TODO
/// - https://rust-random.github.io/book/guide-rngs.html

#[inline]
pub(crate) fn gen_str_id<T: rand::Rng>(size: usize, rng: T) -> String {
    // Use rand::distributions::Alphanumeric as an alternative.
    <Base64 as Distribution<char>>::sample_iter(Base64, rng)
        .take(size)
        .collect()
}

#[derive(Default)]
pub(crate) struct Base64;

impl<T> Distribution<T> for Base64
where
    T: From<u8>,
{
    fn sample<R: rand::Rng + ?Sized>(&self, rng: &mut R) -> T {
        const B64_SIZE: usize = 64;
        const BASE64_CHARSET: &[u8; B64_SIZE] = b"ABCDEFGHIJKLMNOPQRSTUVWXYZ\
                abcdefghijklmnopqrstuvwxyz0123456789-_";
        let ix = rand::distributions::Uniform::new(0, B64_SIZE as u8).sample(rng);
        BASE64_CHARSET[ix as usize].into()
    }
}

#[allow(dead_code)]
fn collision_prob(count: usize, combinations: usize) -> f64 {
    // https://preshing.com/20110504/hash-collision-probabilities/
    let k = count as f64;
    let n = combinations as f64;
    let x = -0.5 * k * (k - 1.0) / n;
    1.0 - x.exp()
}

// calculate the ID size given the current number of IDs and a target collision probability.
pub(crate) fn calc_id_size(n: usize, target: f64) -> usize {
    // Use the approximated general solution to the birthday problem and solve for the number of
    // "days" (d) in the year which is d = 64^(c) where c is the number of characters needed to keep the
    // probability above the target.
    //
    // https://en.wikipedia.org/wiki/Birthday_problem
    //
    // p(n; d) = 1 - e^((-n(n-1)) / 2d)
    // d       = n(n-1) / -2ln(1-p)
    // 64^c    = n(n-1) / -2ln(1-p)
    // c       = log64((n(n-1)) / -2ln(1-p))
    let top = (n * (n - 1)) as f64;
    let bot = -2.0 * (1.0 - target).ln();
    let c = (top / bot).log(64.0).ceil() as usize;
    if c > 2 {
        c
    } else {
        3
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use rand_hc::Hc128Rng;

    #[test]
    fn test_probs() {
        assert_eq!(10, calc_id_size(10_000, 1e-9));
        assert_eq!(7, calc_id_size(1_000, 1e-6));
        assert_eq!(13, calc_id_size(100_000, 1e-12));
        assert_eq!(3, calc_id_size(1, 1e-9));
        assert_eq!(0.999999955000001, 1.0 - collision_prob(10, 1_000_000_000));
        // let mut i = 1;
        // while i < 100_000 {
        //     println!("{} {}", i, calc_id_size(i, 1e-9));
        //     // println!("{}", i);
        //     i += 1000;
        // }
        //
        // for k in [
        //     1,
        //     10,
        //     100,
        //     1_000,
        //     10_000,
        //     1_000_000,
        //     1_200_000,
        //     2_000_000,
        //     9_000_000,
        //     1e7 as usize,
        //     1e8 as usize,
        //     1e9 as usize,
        // ] {
        //     println!("{:10}: {}", k, calc_id_size(k, 1e-10));
        // }
    }

    #[test]
    fn test_gen_id() {
        use rand::SeedableRng;
        let mut rng = Hc128Rng::seed_from_u64(11);
        let id: Vec<u8> = Base64.sample_iter(&mut rng).take(11).collect();
        let str_id = std::str::from_utf8(&id).unwrap();
        assert_eq!("Qevg0z9ws_T", str_id);
        assert_eq!("2M0CDE0c7jsHPy", gen_str_id(14, &mut rng));
        assert_eq!(
            "Qj-ZDcmtPwkANcXs8_dm6Rm9nPtJZ0DWlrKN4IHTXrUj_CXvUrGjIP16nga8FtwW",
            gen_str_id(64, &mut rng)
        );
    }
}
